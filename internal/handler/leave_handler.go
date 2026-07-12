package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kartezya-hr/internal/authz"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type LeaveHandler struct {
	leaveService      service.LeaveService
	employeeService   service.EmployeeService
	emailService      service.EmailService
	mailConfigService service.MailConfigService
}

func NewLeaveHandler(
	leaveService service.LeaveService,
	employeeService service.EmployeeService,
	emailService service.EmailService,
	mailConfigService service.MailConfigService,
) *LeaveHandler {
	return &LeaveHandler{
		leaveService:      leaveService,
		employeeService:   employeeService,
		emailService:      emailService,
		mailConfigService: mailConfigService,
	}
}

type CreateLeaveTypeRequest struct {
	Name               string `json:"name" binding:"required"`
	Description        string `json:"description"`
	IsPaid             bool   `json:"is_paid"`
	LimitAmount        *int   `json:"limit_amount"`
	IsAccrual          bool   `json:"is_accrual"`
	IsRequiredDocument bool   `json:"is_required_document"`
}

type CreateLeaveRequestRequest struct {
	EmployeeID          *uint     `json:"employee_id"`
	LeaveTypeID         uint      `json:"leave_type_id" binding:"required"`
	StartDate           time.Time `json:"start_date" binding:"required"`
	EndDate             time.Time `json:"end_date" binding:"required"`
	IsStartDateFullDay  bool      `json:"is_start_date_full_day" binding:""`
	IsFinishDateFullDay bool      `json:"is_finish_date_full_day" binding:""`
	Reason              string    `json:"reason"`
}

type UpdateLeaveRequestRequest struct {
	LeaveTypeID         uint      `json:"leave_type_id" binding:"required"`
	StartDate           time.Time `json:"start_date" binding:"required"`
	EndDate             time.Time `json:"end_date" binding:"required"`
	IsStartDateFullDay  bool      `json:"is_start_date_full_day" binding:""`
	IsFinishDateFullDay bool      `json:"is_finish_date_full_day" binding:""`
	Reason              string    `json:"reason"`
}

type ApproveRejectRequest struct {
	Reason string `json:"reason,omitempty"`
}

type CancelLeaveRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type CalculateWorkingDaysRequest struct {
	StartDate           time.Time  `json:"start_date" binding:"required"`
	EndDate             *time.Time `json:"end_date"`
	RequestedDays       *float64   `json:"requested_days"`
	IsStartDateFullDay  bool       `json:"is_start_date_full_day"`
	IsFinishDateFullDay bool       `json:"is_finish_date_full_day"`
}

type CalculateWorkingDaysResponse struct {
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	WorkingDays  float64   `json:"working_days"`
	CalendarDays int       `json:"calendar_days"`
}

// CreateLeaveType godoc
// @Summary Create leave type
// @Description Create a new leave type (Admin only)
// @Tags leave-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param leaveType body CreateLeaveTypeRequest true "Leave type data"
// @Success 201 {object} APIResponse{data=types.LeaveTypeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave-types [post]
func (h *LeaveHandler) CreateLeaveType(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	var req CreateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	leaveType := &domain.LeaveType{
		Name:               req.Name,
		Description:        req.Description,
		IsPaid:             req.IsPaid,
		LimitAmount:        req.LimitAmount,
		IsAccrual:          req.IsAccrual,
		IsRequiredDocument: req.IsRequiredDocument,
	}

	if err := h.leaveService.CreateLeaveType(leaveType, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "only administrators can create leave types" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    leaveType,
		"message": "Leave type created successfully",
	})
}

// ListLeaveTypes godoc
// @Summary List leave types
// @Description Get paginated list of leave types (Admin only)
// @Tags leave-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Success 200 {object} APIResponse{data=[]types.LeaveTypeResponse}
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/types [get]
func (h *LeaveHandler) ListLeaveTypes(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse sorting parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	result, err := h.leaveService.GetAllLeaveTypes(page, limit, sortParams)
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

// GetLeaveTypeByID godoc
// @Summary Get leave type by ID
// @Description Get a specific leave type by ID (Admin only)
// @Tags leave-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Type ID"
// @Success 200 {object} APIResponse{data=types.LeaveTypeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /leave/types/{id} [get]
func (h *LeaveHandler) GetLeaveTypeByID(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid leave type ID",
		})
		return
	}

	leaveType, err := h.leaveService.GetLeaveTypeByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Leave type not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    leaveType,
	})
}

// UpdateLeaveType godoc
// @Summary Update leave type
// @Description Update a leave type by ID (Admin only)
// @Tags leave-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Type ID"
// @Param request body CreateLeaveTypeRequest true "Updated leave type data"
// @Success 200 {object} APIResponse{data=types.LeaveTypeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/types/{id} [put]
func (h *LeaveHandler) UpdateLeaveType(c *gin.Context) {
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
			"error":   "Invalid leave type ID",
		})
		return
	}

	var req CreateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	leaveType := &domain.LeaveType{
		Name:               req.Name,
		Description:        req.Description,
		IsPaid:             req.IsPaid,
		LimitAmount:        req.LimitAmount,
		IsAccrual:          req.IsAccrual,
		IsRequiredDocument: req.IsRequiredDocument,
	}

	leaveType.ID = id

	if err := h.leaveService.UpdateLeaveType(leaveType, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "only administrators can update leave types" {
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
		"data":    leaveType,
		"message": "Leave type updated successfully",
	})
}

// DeleteLeaveType godoc
// @Summary Delete leave type
// @Description Delete a leave type by ID (Admin only)
// @Tags leave-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Type ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/types/{id} [delete]
func (h *LeaveHandler) DeleteLeaveType(c *gin.Context) {
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
			"error":   "Invalid leave type ID",
		})
		return
	}

	if err := h.leaveService.DeleteLeaveType(id, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "only administrators can delete leave types" {
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
		"message": "Leave type deleted successfully",
	})
}

// CreateLeaveRequest godoc
// @Summary Create a leave request
// @Description Create a new leave request (Employee for own, Admin for any)
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateLeaveRequestRequest true "Leave request data"
// @Success 201 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/requests [post]
func (h *LeaveHandler) CreateLeaveRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	var req CreateLeaveRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Authorization check: Non-admin users can only create leave requests for themselves
	if !isAdmin(roles) {
		// First get the employee record from userID to get the correct employeeID
		log.Printf("DEBUG: Non-admin user creating leave request - userID: %d, requestedEmployeeID: %v", userID, req.EmployeeID)

		employee, err := h.employeeService.GetEmployeeByUserID(userID)
		if err != nil {
			log.Printf("DEBUG: Failed to get employee by userID %d - Error: %v", userID, err)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Unable to verify employee authorization",
			})
			return
		}

		log.Printf("DEBUG: Employee found - employee.ID: %d, req.EmployeeID: %v", employee.ID, req.EmployeeID)

		if req.EmployeeID == nil {
			req.EmployeeID = &employee.ID
		} else if *req.EmployeeID != employee.ID {
			log.Printf("DEBUG: Authorization failed - Employee mismatch. Employee.ID: %d != req.EmployeeID: %d", employee.ID, *req.EmployeeID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "You can only create leave requests for yourself",
			})
			return
		}

		log.Printf("DEBUG: Authorization passed - Employee ID matches")
	} else if req.EmployeeID == nil {
		// Admin users: if no employee_id provided, use authenticated user's employee ID
		employee, err := h.employeeService.GetEmployeeByUserID(userID)
		if err != nil {
			log.Printf("DEBUG: Admin user - Failed to get employee by userID %d - Error: %v", userID, err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "employee_id is required or authenticated user must be an employee",
			})
			return
		}
		req.EmployeeID = &employee.ID
	}

	// Calculate working days between start and end date
	workingDays, err := h.leaveService.CalculateWorkingDays(req.StartDate, req.EndDate, req.IsStartDateFullDay, req.IsFinishDateFullDay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to calculate working days: " + err.Error(),
		})
		return
	}

	// Fetch the LeaveType to get the IsPaid value
	leaveType, err := h.leaveService.GetLeaveTypeByID(req.LeaveTypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid leave type ID",
		})
		return
	}

	// All leave requests start as PENDING (including admin's own requests)
	// Admin can approve later if needed
	initialStatus := domain.LeaveStatusPending

	// Create LeaveRequest entity
	leave := &domain.LeaveRequest{
		EmployeeID:          *req.EmployeeID,
		LeaveTypeID:         req.LeaveTypeID,
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		IsStartDateFullDay:  req.IsStartDateFullDay,
		IsFinishDateFullDay: req.IsFinishDateFullDay,
		RequestedDays:       workingDays,
		Reason:              req.Reason,
		Status:              initialStatus,
		IsPaid:              leaveType.IsPaid,
	}

	if err := h.leaveService.CreateLeave(leave, userID, isAdmin(roles)); err != nil {
		status := http.StatusInternalServerError
		// Check for balance validation errors
		if strings.Contains(err.Error(), "insufficient leave balance") ||
			strings.Contains(err.Error(), "no leave balance found") ||
			errors.Is(err, service.ErrLeaveTypeLimitExceeded) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// ── INFO notification: INFO_EMAIL_NEW_LEAVE_REQUEST ────────────────────
	go func(empID uint, startDate, endDate time.Time, reason, leaveTypeName string, days float64, isPaid bool) {
		to, cc, bcc, templateCode, notifErr := h.mailConfigService.ResolveRecipients("INFO_EMAIL_NEW_LEAVE_REQUEST")
		if notifErr != nil || len(to) == 0 {
			log.Printf("[LEAVE] INFO_EMAIL_NEW_LEAVE_REQUEST not configured or inactive, skipping notification: %v", notifErr)
			return
		}
		empName := fmt.Sprintf("Çalışan #%d", empID)
		if emp, err := h.employeeService.GetEmployeeByID(empID); err == nil {
			empName = emp.FirstName + " " + emp.LastName
		}
		paidLabel := "Hayır"
		if isPaid {
			paidLabel = "Evet"
		}
		// Kalan yıllık izin bakiyesi
		balanceRow := ""
		if balance, err := h.leaveService.GetAnnualLeaveBalance(empID); err == nil && balance != nil {
			balanceRow = fmt.Sprintf(
				"<tr><td><strong>Yıllık İzin Bakiyesi</strong></td><td>%.1f gün kalan (Toplam: %.1f / Kullanılan: %.1f)</td></tr>",
				balance.RemainingDays, balance.TotalDays, balance.UsedDays,
			)
		}
		body := fmt.Sprintf(
			"<p>Yeni bir izin talebi oluşturuldu.</p>"+
				"<table>"+
				"<tr><td><strong>Çalışan</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>İzin Türü</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Başlangıç</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Bitiş</strong></td><td>%s</td></tr>"+
				"<tr><td><strong>Gün Sayısı</strong></td><td>%.1f iş günü</td></tr>"+
				"<tr><td><strong>Ücretli İzin</strong></td><td>%s</td></tr>"+
				"%s"+
				"<tr><td><strong>Açıklama</strong></td><td>%s</td></tr>"+
				"</table>",
			empName, leaveTypeName,
			startDate.Format("02.01.2006"),
			endDate.Format("02.01.2006"),
			days, paidLabel, balanceRow, reason,
		)
		vars := map[string]interface{}{"body": body}
		if err := h.emailService.SendTemplateEmailWithCC(to, cc, bcc, "Yeni İzin Talebi", templateCode, vars); err != nil {
			log.Printf("[LEAVE] INFO notification error (INFO_EMAIL_NEW_LEAVE_REQUEST): %v", err)
		} else {
			log.Printf("[LEAVE] INFO notification sent for new leave request (employee %d)", empID)
		}
	}(*req.EmployeeID, req.StartDate, req.EndDate, req.Reason, leaveType.Name, workingDays, leave.IsPaid)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Leave request created successfully",
	})
}

// UpdateLeaveRequest godoc
// @Summary Update leave request
// @Description Update a leave request by ID (Employee for own, Admin for any)
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Request ID"
// @Param request body UpdateLeaveRequestRequest true "Updated leave request data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/requests/{id} [put]
func (h *LeaveHandler) UpdateLeaveRequest(c *gin.Context) {
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
			"error":   "Invalid leave request ID",
		})
		return
	}

	var req UpdateLeaveRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get existing leave request for authorization check
	existingLeave, err := h.leaveService.GetLeaveByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Leave request not found",
		})
		return
	}

	// Authorization check - employees can only update their own requests, admins can update any
	if !isAdmin(roles) {
		// First get the employee record from userID to get the correct employeeID
		employee, err := h.employeeService.GetEmployeeByUserID(userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Unable to verify employee authorization",
			})
			return
		}

		// Log the values for debugging authorization
		log.Printf("Authorization check - existingLeave.EmployeeID: %d, employee.ID: %d", existingLeave.EmployeeID, employee.ID)

		if existingLeave.EmployeeID != employee.ID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "You can only update your own leave requests",
			})
			return
		}
	}

	// Calculate working days between start and end date
	workingDays, err := h.leaveService.CalculateWorkingDays(req.StartDate, req.EndDate, req.IsStartDateFullDay, req.IsFinishDateFullDay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to calculate working days: " + err.Error(),
		})
		return
	}

	// Fetch the LeaveType to get the IsPaid value
	leaveType, err := h.leaveService.GetLeaveTypeByID(req.LeaveTypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid leave type ID",
		})
		return
	}

	// Create updated LeaveRequest entity
	leave := &domain.LeaveRequest{
		EmployeeID:          existingLeave.EmployeeID,
		LeaveTypeID:         req.LeaveTypeID,
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		IsStartDateFullDay:  req.IsStartDateFullDay,
		IsFinishDateFullDay: req.IsFinishDateFullDay,
		RequestedDays:       workingDays,
		Reason:              existingLeave.Reason, // Preserve existing reason initially
		Status:              existingLeave.Status, // Preserve existing status
		IsPaid:              leaveType.IsPaid,
	}
	leave.Reason = req.Reason
	leave.ID = id

	if err := h.leaveService.UpdateLeave(leave, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "only pending leave requests can be updated, current status: APPROVED" ||
			err.Error() == "only pending leave requests can be updated, current status: REJECTED" ||
			err.Error() == "only pending leave requests can be updated, current status: CANCELLED" {
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
		"message": "Leave request updated successfully",
	})
}

// GetLeaveRequestByID godoc
// @Summary Get leave request by ID
// @Description Get a specific leave request by ID (Admin only)
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Request ID"
// @Success 200 {object} APIResponse{data=types.AdminLeaveRequestResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /leave/requests/{id} [get]
func (h *LeaveHandler) GetLeaveRequestByID(c *gin.Context) {
	_, _, _, ok := getUserContext(c)
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
			"error":   "Invalid leave request ID",
		})
		return
	}

	leave, err := h.leaveService.GetLeaveByIDFormatted(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Leave request not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    leave,
	})
}

// GetMyLeaveRequests godoc
// @Summary Get my leave requests
// @Description Get paginated leave requests for the authenticated employee
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param status query string false "Filter by status (PENDING, APPROVED, REJECTED)"
// @Param sort query string false "Sort field (default: created_at)"
// @Param direction query string false "Sort direction (default: DESC)"
// @Success 200 {object} APIResponse{data=[]types.MyLeaveRequestResponse}
// @Failure 401 {object} APIResponse
// @Router /leave/requests/me [get]
func (h *LeaveHandler) GetMyLeaveRequests(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status") // Optional status filter: PENDING, APPROVED, REJECTED

	var leaveTypeID *uint
	if layTypeIDStr := c.Query("leave_type_id"); layTypeIDStr != "" {
		if layTypeID, err := strconv.ParseUint(layTypeIDStr, 10, 32); err == nil {
			id := uint(layTypeID)
			leaveTypeID = &id
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

	// Parse sorting parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "created_at"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	result, err := h.leaveService.GetMyLeaveRequestsPaginated(userID, page, limit, sortParams, status, leaveTypeID, startDate, endDate)
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

// GetAllLeaveRequests godoc
// @Summary Get all leave requests
// @Description Get paginated list of all leave requests (Admin only)
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param employee_id query int false "Filter by employee ID"
// @Param status query string false "Filter by status (PENDING, APPROVED, REJECTED)"
// @Param sort query string false "Sort field (default: created_at)"
// @Param direction query string false "Sort direction (default: DESC)"
// @Success 200 {object} APIResponse{data=[]types.AdminLeaveRequestResponse}
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/requests [get]
func (h *LeaveHandler) GetAllLeaveRequests(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanViewLeaveManagement) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only administrators can view all leave requests",
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status") // Optional status filter: PENDING, APPROVED, REJECTED

	var empIDPtr *uint
	employeeIDStr := c.Query("employee_id")
	if employeeIDStr != "" {
		employeeID, err := strconv.ParseUint(employeeIDStr, 10, 32)
		if err == nil {
			id := uint(employeeID)
			empIDPtr = &id
		}
	}

	var leaveTypeID *uint
	if layTypeIDStr := c.Query("leave_type_id"); layTypeIDStr != "" {
		if layTypeID, err := strconv.ParseUint(layTypeIDStr, 10, 32); err == nil {
			id := uint(layTypeID)
			leaveTypeID = &id
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

	// Parse sorting parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "created_at"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	result, err := h.leaveService.GetAllLeaveRequestsPaginated(empIDPtr, page, limit, sortParams, status, leaveTypeID, startDate, endDate)
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

func (h *LeaveHandler) GetPendingLeaveRequests(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !isAdmin(roles) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only administrators can view pending leave requests",
		})
		return
	}

	// Use existing service method
	leaves, err := h.leaveService.GetLeavesByStatus(domain.LeaveStatusPending, "created_at", "DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    leaves,
	})
}

// ApproveLeaveRequest godoc
// @Summary Approve leave request
// @Description Approve a leave request by ID (Admin only)
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Request ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/requests/{id}/approve [post]
func (h *LeaveHandler) ApproveLeaveRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanApproveLeave) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only administrators can approve leave requests",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid leave request ID",
		})
		return
	}

	if err := h.leaveService.ApproveLeave(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Leave request approved successfully",
	})
}

// RejectLeaveRequest godoc
// @Summary Reject leave request
// @Description Reject a leave request by ID (Admin only)
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Request ID"
// @Param request body ApproveRejectRequest true "Rejection reason"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/requests/{id}/reject [post]
func (h *LeaveHandler) RejectLeaveRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanApproveLeave) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only administrators can reject leave requests",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid leave request ID",
		})
		return
	}

	var req ApproveRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := h.leaveService.RejectLeave(id, req.Reason, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Leave request rejected successfully",
	})
}

// CancelLeaveRequest godoc
// @Summary Cancel leave request
// @Description Cancel a leave request by ID (Employee for own, Admin for any)
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Request ID"
// @Param request body CancelLeaveRequest true "Cancellation reason"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/requests/{id}/cancel [post]
func (h *LeaveHandler) CancelLeaveRequest(c *gin.Context) {
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
			"error":   "Invalid leave request ID",
		})
		return
	}

	if err := h.leaveService.CancelLeave(id, "İptal", userID, hasCapability(roles, authz.CanApproveLeave)); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "you can only cancel your own leave requests" ||
			err.Error() == "only pending leave requests can be cancelled, current status: PENDING" ||
			err.Error() == "only pending leave requests can be cancelled, current status: APPROVED" ||
			err.Error() == "only pending leave requests can be cancelled, current status: REJECTED" ||
			err.Error() == "only pending leave requests can be cancelled, current status: CANCELLED" {
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
		"message": "Leave request cancelled successfully",
	})
}

// GetMyLeaveBalances godoc
// @Summary Get my leave balances
// @Description Get paginated leave balances for the authenticated employee
// @Tags leave-balances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param sort query string false "Sort field" default("leave_type_id")
// @Param direction query string false "Sort direction (ASC/DESC)" default("ASC")
// @Success 200 {object} map[string]interface{} "success: true, data: []MyLeaveBalanceResponse, page: PageInfo"
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /leave/balances/me [get]
func (h *LeaveHandler) GetMyLeaveBalances(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse sorting parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "leave_type_id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	result, err := h.leaveService.GetMyLeaveBalances(userID, page, limit, sortParams)
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

// GetLeaveBalances godoc
// @Summary Get employee leave balances
// @Description Get paginated leave balances for a specific employee (Admin only)
// @Tags leave-balances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param employee_id query int true "Employee ID"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: leave_type_id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Success 200 {object} APIResponse{data=[]types.MyLeaveBalanceResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /leave/balances [get]
func (h *LeaveHandler) GetLeaveBalances(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !hasCapability(roles, authz.CanManageLeaveTypes) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only administrators can view other employees' leave balances",
		})
		return
	}

	employeeIDStr := c.Query("employee_id")
	if employeeIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "employee_id query parameter is required",
		})
		return
	}

	employeeID, err := strconv.ParseUint(employeeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid employee_id",
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse sorting parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "leave_type_id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	result, err := h.leaveService.GetEmployeeLeaveBalances(uint(employeeID), page, limit, sortParams)
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

// CalculateWorkingDays godoc
// @Summary Calculate working days between two dates
// @Description Calculate the number of working days (excluding weekends and holidays) between two dates
// @Tags leave-requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CalculateWorkingDaysRequest true "Date range for calculation"
// @Success 200 {object} APIResponse{data=CalculateWorkingDaysResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /leave/calculate-working-days [post]
func (h *LeaveHandler) CalculateWorkingDays(c *gin.Context) {
	_, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	var req CalculateWorkingDaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate inputs
	if req.EndDate == nil && req.RequestedDays == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Either end_date or requested_days must be provided",
		})
		return
	}

	var workingDays float64
	var endDate time.Time
	var err error

	if req.EndDate != nil {
		endDate = *req.EndDate
		if req.StartDate.After(endDate) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Start date must be before or equal to end date",
			})
			return
		}

		// Calculate working days using service
		workingDays, err = h.leaveService.CalculateWorkingDays(req.StartDate, endDate, req.IsStartDateFullDay, req.IsFinishDateFullDay)
	} else if req.RequestedDays != nil {
		workingDays = *req.RequestedDays
		endDate, err = h.leaveService.CalculateEndDate(req.StartDate, workingDays, req.IsStartDateFullDay, req.IsFinishDateFullDay)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Calculate calendar days
	calendarDays := int(endDate.Sub(req.StartDate).Hours()/24) + 1

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": CalculateWorkingDaysResponse{
			StartDate:    req.StartDate,
			EndDate:      endDate,
			WorkingDays:  workingDays,
			CalendarDays: calendarDays,
		},
	})
}

// ==================== Leave Document Handlers ====================

// UploadLeaveDocument godoc
// @Summary Upload leave document
// @Description Upload a medical report or other document for a leave request
// @Tags leave-documents
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Request ID"
// @Param file formData file true "Document file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /leave/requests/{id}/documents [post]
func (h *LeaveHandler) UploadLeaveDocument(c *gin.Context) {
	// Get user from context
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get leave request ID from URL
	idParam := c.Param("id")
	leaveRequestID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid leave request ID",
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
	document, err := h.leaveService.UploadLeaveDocument(uint(leaveRequestID), file, userID)
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

// GetLeaveDocuments godoc
// @Summary Get leave documents
// @Description Get all documents for a leave request
// @Tags leave-documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Request ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /leave/requests/{id}/documents [get]
func (h *LeaveHandler) GetLeaveDocuments(c *gin.Context) {
	// Get user from context
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Get leave request ID from URL
	idParam := c.Param("id")
	leaveRequestID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid leave request ID",
		})
		return
	}

	// Get documents
	documents, err := h.leaveService.GetLeaveDocuments(uint(leaveRequestID), userID, hasCapability(roles, authz.CanViewLeaveManagement))
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

// DeleteLeaveDocument godoc
// @Summary Delete leave document
// @Description Delete a document from a leave request
// @Tags leave-documents
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID (UUID)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /leave/documents/{id} [delete]
func (h *LeaveHandler) DeleteLeaveDocument(c *gin.Context) {
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
	if err := h.leaveService.DeleteLeaveDocument(documentID, userID, hasCapability(roles, authz.CanViewLeaveManagement)); err != nil {
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

// DownloadLeaveDocument godoc
// @Summary Download leave document
// @Description Get download URL for a leave document
// @Tags leave-documents
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID (UUID)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /leave/documents/{id}/download [get]
func (h *LeaveHandler) DownloadLeaveDocument(c *gin.Context) {
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
	url, err := h.leaveService.DownloadLeaveDocument(documentID, userID, hasCapability(roles, authz.CanViewLeaveManagement))
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
