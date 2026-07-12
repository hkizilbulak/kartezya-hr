package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"kartezya-hr/internal/authz"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type ExpenseHandler struct {
	expenseService    service.ExpenseService
	employeeService   service.EmployeeService
	emailService      service.EmailService
	mailConfigService service.MailConfigService
}

// CreateExpenseRequestDTO represents the DTO for creating expense requests
type CreateExpenseRequestDTO struct {
	ExpenseTypeID uint    `json:"expense_type_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Currency      string  `json:"currency" binding:"required,oneof=TRY USD EUR"`
	ExpenseDate   string  `json:"expense_date" binding:"required"` // Date string in YYYY-MM-DD format
	Description   string  `json:"description" binding:"required"`
}

// UpdateExpenseRequestDTO represents the DTO for updating expense requests
type UpdateExpenseRequestDTO struct {
	ExpenseTypeID *uint    `json:"expense_type_id"`
	Amount        *float64 `json:"amount" binding:"omitempty,gt=0"`
	Currency      *string  `json:"currency" binding:"omitempty,oneof=TRY USD EUR"`
	ExpenseDate   *string  `json:"expense_date"` // Date string in YYYY-MM-DD format
	Description   *string  `json:"description"`
}

// CreateExpenseTypeRequestDTO represents DTO for creating expense type
type CreateExpenseTypeRequestDTO struct {
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	RequiresReceipt bool     `json:"requires_receipt"`
	MaxAmount       *float64 `json:"max_amount"`
	Active          bool     `json:"active"`
	RoleID          *uint    `json:"role_id"`
}

// UpdateExpenseTypeRequestDTO represents DTO for updating expense type
type UpdateExpenseTypeRequestDTO struct {
	Name            *string         `json:"name"`
	Description     *string         `json:"description"`
	RequiresReceipt *bool           `json:"requires_receipt"`
	MaxAmount       json.RawMessage `json:"max_amount" swaggertype:"number" example:"100.0"` // Use null to clear, omit to keep existing
	Active          *bool           `json:"active"`
	RoleID          json.RawMessage `json:"role_id" swaggertype:"integer" example:"1"` // Use null to clear, omit to keep existing
}

func NewExpenseHandler(
	expenseService service.ExpenseService,
	employeeService service.EmployeeService,
	emailService service.EmailService,
	mailConfigService service.MailConfigService,
) *ExpenseHandler {
	return &ExpenseHandler{
		expenseService:    expenseService,
		employeeService:   employeeService,
		emailService:      emailService,
		mailConfigService: mailConfigService,
	}
}

// CreateExpenseRequest godoc
// @Summary Create expense request
// @Description Create a new expense request
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateExpenseRequestDTO true "Expense request data"
// @Success 201 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /expense/requests [post]
func (h *ExpenseHandler) CreateExpenseRequest(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	var dto CreateExpenseRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Parse expense date from string (YYYY-MM-DD) to time.Time
	expenseDate, err := time.Parse("2006-01-02", dto.ExpenseDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense_date format. Expected YYYY-MM-DD",
		})
		return
	}

	// Create expense request model
	expense := domain.ExpenseRequest{
		ExpenseTypeID: dto.ExpenseTypeID,
		Amount:        dto.Amount,
		Currency:      dto.Currency,
		ExpenseDate:   expenseDate,
		Description:   dto.Description,
	}

	if err := h.expenseService.CreateExpenseRequest(&expense, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// ── INFO notification: INFO_EMAIL_NEW_EXPENSE_REQUEST ─────────────────
	go func(expID uint, empID uint, expTypeID uint, amount float64, currency, description string, expDate time.Time) {
		to, cc, bcc, templateCode, notifErr := h.mailConfigService.ResolveRecipients("INFO_EMAIL_NEW_EXPENSE_REQUEST")
		if notifErr != nil || len(to) == 0 {
			log.Printf("[EXPENSE] INFO_EMAIL_NEW_EXPENSE_REQUEST not configured or inactive, skipping notification: %v", notifErr)
			return
		}
		empName := fmt.Sprintf("Çalışan #%d", empID)
		if emp, err := h.employeeService.GetEmployeeByID(empID); err == nil {
			empName = emp.FirstName + " " + emp.LastName
		}
		expTypeName := fmt.Sprintf("#%d", expTypeID)
		if et, err := h.expenseService.GetExpenseTypeByID(expTypeID); err == nil {
			expTypeName = et.Name
		}
		body := fmt.Sprintf(
			"<p>Yeni bir masraf talebi oluşturuldu.</p>"+
				"<table>"+
				"<tr><td><strong>Masraf No</strong></td><td>#%d</td></tr>"+
				"<tr><td><strong>Çalışan</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Masraf Türü</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Tutar</strong></td><td>%.2f %s</td></tr>"+
				"<tr><td><strong>Tarih</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Açıklama</strong></td><td>%s</td></tr>"+
				"</table>",
			expID, empName, expTypeName,
			amount, currency,
			expDate.Format("02.01.2006"),
			description,
		)
		vars := map[string]interface{}{"body": body}
		if err := h.emailService.SendTemplateEmailWithCC(to, cc, bcc, "Yeni Masraf Talebi", templateCode, vars); err != nil {
			log.Printf("[EXPENSE] INFO notification error (INFO_EMAIL_NEW_EXPENSE_REQUEST): %v", err)
		} else {
			log.Printf("[EXPENSE] INFO notification sent for new expense request #%d", expID)
		}
	}(expense.ID, expense.EmployeeID, expense.ExpenseTypeID, expense.Amount, expense.Currency, expense.Description, expense.ExpenseDate)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    expense,
		"message": "Expense request created successfully",
	})
}

// GetMyExpenseRequests godoc
// @Summary Get my expense requests
// @Description Get paginated expense requests for the authenticated employee
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param status query string false "Filter by status (PENDING, APPROVED, REJECTED, PAID)"
// @Param sort query string false "Sort field (default: created_at)"
// @Param direction query string false "Sort direction (default: DESC)"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /expense/requests/me [get]
func (h *ExpenseHandler) GetMyExpenseRequests(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	var expenseTypeID *uint
	if extTypeIDStr := c.Query("expense_type_id"); extTypeIDStr != "" {
		if extTypeID, err := strconv.ParseUint(extTypeIDStr, 10, 32); err == nil {
			id := uint(extTypeID)
			expenseTypeID = &id
		}
	}

	var startDate *string
	if start := c.Query("start_date"); start != "" {
		startDate = &start
	}

	var endDate *string
	if end := c.Query("end_date"); end != "" {
		endDate = &end
	}

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "created_at"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	result, err := h.expenseService.GetMyExpenseRequestsPaginated(userID, page, limit, sortParams, status, expenseTypeID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Data,
		"page":    result.Page,
	})
}

// GetAllExpenseRequests godoc
// @Summary Get all expense requests
// @Description Get paginated list of all expense requests (Admin only)
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param employee_id query int false "Filter by employee ID"
// @Param status query string false "Filter by status (PENDING, APPROVED, REJECTED, PAID)"
// @Param sort query string false "Sort field (default: created_at)"
// @Param direction query string false "Sort direction (default: DESC)"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /expense/requests [get]
func (h *ExpenseHandler) GetAllExpenseRequests(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanViewExpenseManagement) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	var employeeID *uint
	if empIDStr := c.Query("employee_id"); empIDStr != "" {
		if empID, err := strconv.ParseUint(empIDStr, 10, 32); err == nil {
			id := uint(empID)
			employeeID = &id
		}
	}

	var expenseTypeID *uint
	if extTypeIDStr := c.Query("expense_type_id"); extTypeIDStr != "" {
		if extTypeID, err := strconv.ParseUint(extTypeIDStr, 10, 32); err == nil {
			id := uint(extTypeID)
			expenseTypeID = &id
		}
	}

	var startDate *string
	if start := c.Query("start_date"); start != "" {
		startDate = &start
	}

	var endDate *string
	if end := c.Query("end_date"); end != "" {
		endDate = &end
	}

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "created_at"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	result, err := h.expenseService.GetAllExpenseRequestsPaginated(employeeID, page, limit, sortParams, status, expenseTypeID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Data,
		"page":    result.Page,
	})
}

// GetExpenseRequestByID godoc
// @Summary Get expense request by ID
// @Description Get expense request details by ID
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /expense/requests/{id} [get]
func (h *ExpenseHandler) GetExpenseRequestByID(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	canViewManagement := hasCapability(roles, authz.CanViewExpenseManagement)
	expense, err := h.expenseService.GetExpenseRequestByIDForCaller(id, userID, canViewManagement)
	if err != nil {
		status := http.StatusNotFound
		if err.Error() == "access denied" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    expense,
	})
}

// UpdateExpenseRequest godoc
// @Summary Update expense request
// @Description Update an existing expense request (only pending requests)
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Param request body domain.ExpenseRequest true "Expense request data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /expense/requests/{id} [put]
func (h *ExpenseHandler) UpdateExpenseRequest(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	var dto UpdateExpenseRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get existing expense request
	existing, err := h.expenseService.GetExpenseRequestByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Verify ownership
	if existing.Employee != nil && existing.Employee.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Access denied",
		})
		return
	}

	// Update fields if provided
	if dto.ExpenseTypeID != nil {
		existing.ExpenseTypeID = *dto.ExpenseTypeID
	}
	if dto.Amount != nil {
		existing.Amount = *dto.Amount
	}
	if dto.Currency != nil {
		existing.Currency = *dto.Currency
	}
	if dto.ExpenseDate != nil {
		expenseDate, err := time.Parse("2006-01-02", *dto.ExpenseDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid expense_date format. Expected YYYY-MM-DD",
			})
			return
		}
		existing.ExpenseDate = expenseDate
	}
	if dto.Description != nil {
		existing.Description = *dto.Description
	}

	if err := h.expenseService.UpdateExpenseRequest(existing, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    existing,
		"message": "Expense request updated successfully",
	})
}

// DeleteExpenseRequest godoc
// @Summary Delete expense request
// @Description Delete an expense request (only pending requests)
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /expense/requests/{id} [delete]
func (h *ExpenseHandler) DeleteExpenseRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	if err := h.expenseService.DeleteExpenseRequest(id, userID, isAdmin(roles)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Expense request deleted successfully",
	})
}

// ApproveExpenseRequest godoc
// @Summary Approve expense request
// @Description Approve an expense request (Admin only)
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /expense/requests/{id}/approve [post]
func (h *ExpenseHandler) ApproveExpenseRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanApproveExpense) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	if err := h.expenseService.ApproveExpenseRequest(id, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// ── INFO notification: INFO_EMAIL_EXPENSE_APPROVED ────────────────────
	go func(expID uint) {
		to, cc, bcc, templateCode, notifErr := h.mailConfigService.ResolveRecipients("INFO_EMAIL_EXPENSE_APPROVED")
		if notifErr != nil || len(to) == 0 {
			log.Printf("[EXPENSE] INFO_EMAIL_EXPENSE_APPROVED not configured or inactive, skipping notification: %v", notifErr)
			return
		}
		exp, err := h.expenseService.GetExpenseRequestByID(expID)
		if err != nil {
			log.Printf("[EXPENSE] Could not fetch expense #%d for INFO_EMAIL_EXPENSE_APPROVED: %v", expID, err)
			return
		}
		empName := fmt.Sprintf("Çalışan #%d", exp.EmployeeID)
		if exp.Employee != nil {
			empName = exp.Employee.FirstName + " " + exp.Employee.LastName
		}
		expTypeName := fmt.Sprintf("#%d", exp.ExpenseTypeID)
		if exp.ExpenseType != nil {
			expTypeName = exp.ExpenseType.Name
		}
		approverRow := ""
		if exp.Approver != nil {
			approverName := exp.Approver.Email
			approverRow = fmt.Sprintf("<tr><td><strong>Onaylayan</strong></td><td>%s</td></tr>", approverName)
		}
		approvedAtRow := ""
		if exp.ApprovedAt != nil {
			approvedAtRow = fmt.Sprintf("<tr><td><strong>Onay Tarihi</strong></td><td>%s</td></tr>", exp.ApprovedAt.Format("02.01.2006 15:04"))
		}
		body := fmt.Sprintf(
			"<p>Bir masraf talebi onaylandı.</p>"+
				"<table>"+
				"<tr><td><strong>Masraf No</strong></td><td>#%d</td></tr>"+
				"<tr><td><strong>Çalışan</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Masraf Türü</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Tutar</strong></td><td>%.2f %s</td></tr>"+
				"<tr><td><strong>Masraf Tarihi</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Açıklama</strong></td><td>%s</td></tr>"+
				"%s%s"+
				"<tr><td><strong>Durum</strong></td><td><strong style=\"color:#16a34a\">ONAYLANDI</strong></td></tr>"+
				"</table>",
			exp.ID, empName, expTypeName,
			exp.Amount, exp.Currency,
			exp.ExpenseDate.Format("02.01.2006"),
			exp.Description,
			approverRow, approvedAtRow,
		)
		vars := map[string]interface{}{"body": body}
		if err := h.emailService.SendTemplateEmailWithCC(to, cc, bcc, "Masraf Talebiniz Onaylandı", templateCode, vars); err != nil {
			log.Printf("[EXPENSE] INFO notification error (INFO_EMAIL_EXPENSE_APPROVED): %v", err)
		} else {
			log.Printf("[EXPENSE] INFO notification sent for approved expense #%d", expID)
		}
	}(id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Expense request approved successfully",
	})
}

// RejectExpenseRequest godoc
// @Summary Reject expense request
// @Description Reject an expense request (Admin only)
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Param request body map[string]string true "Rejection reason"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /expense/requests/{id}/reject [post]
func (h *ExpenseHandler) RejectExpenseRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanApproveExpense) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	var req struct {
		RejectionReason string `json:"rejection_reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Rejection reason is required",
		})
		return
	}

	if err := h.expenseService.RejectExpenseRequest(id, req.RejectionReason, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Expense request rejected successfully",
	})
}

// MarkExpenseAsPaid godoc
// @Summary Mark expense as paid
// @Description Mark an approved expense request as paid (Admin only)
// @Tags expense-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Param request body map[string]string true "Payment reference"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /expense/requests/{id}/pay [post]
func (h *ExpenseHandler) MarkExpenseAsPaid(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanPayExpense) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	var req struct {
		PaymentReference string `json:"payment_reference"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := h.expenseService.MarkAsPaid(id, req.PaymentReference, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Expense request marked as paid successfully",
	})
}

// Expense Type Handlers

// GetExpenseTypes godoc
// @Summary Get expense types
// @Description Get all expense types with pagination (Admin only)
// @Tags expense-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param role_id query int false "Role ID to filter by"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /expense/types [get]
func (h *ExpenseHandler) GetExpenseTypes(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanManageExpenseTypes) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Insufficient permissions",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "name"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	var roleID *uint
	if roleIDStr := c.Query("role_id"); roleIDStr != "" {
		if id, err := strconv.ParseUint(roleIDStr, 10, 32); err == nil {
			idUint := uint(id)
			roleID = &idUint
		}
	}

	result, err := h.expenseService.GetAllExpenseTypes(page, limit, sortParams, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Data,
		"page":    result.Page,
	})
}

// GetActiveExpenseTypes godoc
// @Summary Get active expense types
// @Description Get all active expense types (for dropdown)
// @Tags expense-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /expense/types/active [get]
func (h *ExpenseHandler) GetActiveExpenseTypes(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	expenseTypes, err := h.expenseService.GetActiveExpenseTypes(roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    expenseTypes,
	})
}

// CreateExpenseType godoc
// @Summary Create expense type
// @Description Create a new expense type (Admin only)
// @Tags expense-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateExpenseTypeRequestDTO true "Expense type data"
// @Success 201 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /expense/types [post]
func (h *ExpenseHandler) CreateExpenseType(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanManageExpenseTypes) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Insufficient permissions",
		})
		return
	}

	var dto CreateExpenseTypeRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	expenseType := domain.ExpenseType{
		Name:            dto.Name,
		Description:     dto.Description,
		RequiresReceipt: dto.RequiresReceipt,
		MaxAmount:       dto.MaxAmount,
		Active:          dto.Active,
		RoleID:          dto.RoleID,
	}

	if err := h.expenseService.CreateExpenseType(&expenseType, strconv.FormatUint(uint64(userID), 10)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    expenseType,
	})
}

// UpdateExpenseType godoc
// @Summary Update expense type
// @Description Update an existing expense type (Admin only)
// @Tags expense-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Type ID"
// @Param request body UpdateExpenseTypeRequestDTO true "Expense type update data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /expense/types/{id} [put]
func (h *ExpenseHandler) UpdateExpenseType(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanManageExpenseTypes) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Insufficient permissions",
		})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid ID",
		})
		return
	}

	var dto UpdateExpenseTypeRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get existing
	expenseType, err := h.expenseService.GetExpenseTypeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Update fields
	if dto.Name != nil {
		expenseType.Name = *dto.Name
	}
	if dto.Description != nil {
		expenseType.Description = *dto.Description
	}
	if dto.RequiresReceipt != nil {
		expenseType.RequiresReceipt = *dto.RequiresReceipt
	}
	// MaxAmount: nil json.RawMessage = field absent (no change)
	// "null" = explicitly clear max_amount
	// number = set to that value
	if len(dto.MaxAmount) > 0 {
		if string(dto.MaxAmount) == "null" {
			expenseType.MaxAmount = nil
		} else {
			var maxAmt float64
			if err := json.Unmarshal(dto.MaxAmount, &maxAmt); err == nil {
				if maxAmt <= 0 {
					expenseType.MaxAmount = nil
				} else {
					expenseType.MaxAmount = &maxAmt
				}
			}
		}
	}
	if dto.Active != nil {
		expenseType.Active = *dto.Active
	}
	// RoleID: nil json.RawMessage = field absent (no change)
	// "null" = explicitly set to null → clear role
	// number = set to that role
	if len(dto.RoleID) > 0 {
		if string(dto.RoleID) == "null" {
			expenseType.RoleID = nil
		} else {
			var roleID uint
			if err := json.Unmarshal(dto.RoleID, &roleID); err == nil {
				if roleID == 0 {
					expenseType.RoleID = nil
				} else {
					expenseType.RoleID = &roleID
				}
			}
		}
	}

	if err := h.expenseService.UpdateExpenseType(expenseType, strconv.FormatUint(uint64(userID), 10)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    expenseType,
	})
}

// DeleteExpenseType godoc
// @Summary Delete expense type
// @Description Delete an expense type (Admin only)
// @Tags expense-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Type ID"
// @Success 200 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /expense/types/{id} [delete]
func (h *ExpenseHandler) DeleteExpenseType(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanManageExpenseTypes) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Insufficient permissions",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense type ID",
		})
		return
	}

	if err := h.expenseService.DeleteExpenseType(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Expense type deleted successfully",
	})
}

// ==================== Expense Document Handlers ====================

// UploadExpenseDocument godoc
// @Summary Upload expense document
// @Description Upload a receipt/invoice for an expense request
// @Tags expense-documents
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Param file formData file true "Document file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /expense/requests/{id}/documents [post]
func (h *ExpenseHandler) UploadExpenseDocument(c *gin.Context) {
	// Get user from context
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get expense request ID from URL
	idParam := c.Param("id")
	expenseRequestID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No file uploaded",
		})
		return
	}

	// Upload document
	document, err := h.expenseService.UploadExpenseDocument(uint(expenseRequestID), file, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document uploaded successfully",
		"data":    document,
	})
}

// GetExpenseDocuments godoc
// @Summary Get expense documents
// @Description Get all documents for an expense request
// @Tags expense-documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Expense Request ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /expense/requests/{id}/documents [get]
func (h *ExpenseHandler) GetExpenseDocuments(c *gin.Context) {
	// Get user from context
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get expense request ID from URL
	idParam := c.Param("id")
	expenseRequestID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid expense request ID",
		})
		return
	}

	// Get documents
	documents, err := h.expenseService.GetExpenseDocuments(uint(expenseRequestID), userID, hasCapability(roles, authz.CanViewExpenseManagement))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    documents,
	})
}

// DeleteExpenseDocument godoc
// @Summary Delete expense document
// @Description Delete a document from an expense request
// @Tags expense-documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /expense/documents/{id} [delete]
func (h *ExpenseHandler) DeleteExpenseDocument(c *gin.Context) {
	// Get user from context
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get document ID from URL (UUID string)
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid document ID",
		})
		return
	}

	// Delete document
	if err := h.expenseService.DeleteExpenseDocument(documentID, userID, isAdmin(roles)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document deleted successfully",
	})
}

// DownloadExpenseDocument godoc
// @Summary Download expense document
// @Description Get download URL for an expense document
// @Tags expense-documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Document ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /expense/documents/{id}/download [get]
func (h *ExpenseHandler) DownloadExpenseDocument(c *gin.Context) {
	// Get user from context
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get document ID from URL (UUID string)
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid document ID",
		})
		return
	}

	// Get download URL
	url, err := h.expenseService.DownloadExpenseDocument(documentID, userID, hasCapability(roles, authz.CanViewExpenseManagement))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url": url,
		},
	})
}
