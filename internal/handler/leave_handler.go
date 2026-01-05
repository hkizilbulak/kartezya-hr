package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type LeaveHandler struct {
	leaveService    service.LeaveService
	employeeService service.EmployeeService
}

func NewLeaveHandler(leaveService service.LeaveService, employeeService service.EmployeeService) *LeaveHandler {
	return &LeaveHandler{
		leaveService:    leaveService,
		employeeService: employeeService,
	}
}

type CreateLeaveTypeRequest struct {
	Name               string `json:"name" binding:"required"`
	Description        string `json:"description"`
	IsPaid             bool   `json:"is_paid"`
	IsLimited          bool   `json:"is_limited"`
	IsAccrual          bool   `json:"is_accrual"`
	IsRequiredDocument bool   `json:"is_required_document"`
}

type CreateLeaveRequest struct {
	LeaveTypeID uint      `json:"leave_type_id" binding:"required"`
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	Reason      string    `json:"reason" binding:"required"`
	IsPaid      bool      `json:"is_paid"`
}

type CreateLeaveRequestRequest struct {
	EmployeeID  uint      `json:"employee_id" binding:"required"`
	LeaveTypeID uint      `json:"leave_type_id" binding:"required"`
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	Reason      string    `json:"reason" binding:"required"`
	IsPaid      bool      `json:"is_paid"`
}

type UpdateLeaveRequestRequest struct {
	EmployeeID  uint      `json:"employee_id" binding:"required"`
	LeaveTypeID uint      `json:"leave_type_id" binding:"required"`
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	Reason      string    `json:"reason" binding:"required"`
	IsPaid      bool      `json:"is_paid"`
}

type ApproveRejectRequest struct {
	Reason string `json:"reason,omitempty"`
}

type CancelLeaveRequest struct {
	Reason string `json:"reason" binding:"required"`
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
	_, email, _, ok := getUserContext(c)
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
		IsLimited:          req.IsLimited,
		IsAccrual:          req.IsAccrual,
		IsRequiredDocument: req.IsRequiredDocument,
	}

	if err := h.leaveService.CreateLeaveType(leaveType, email); err != nil {
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

// GetLeaveTypesLookup godoc
// @Summary Get leave types lookup
// @Description Get simplified list of leave types for dropdown/lookup
// @Tags leave-types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=[]types.LeaveTypeLookup}
// @Failure 500 {object} APIResponse
// @Router /leave/types/lookup [get]
func (h *LeaveHandler) GetLeaveTypesLookup(c *gin.Context) {
	leaveTypes, err := h.leaveService.GetLeaveTypesLookup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    leaveTypes,
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
	_, email, _, ok := getUserContext(c)
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
		IsLimited:          req.IsLimited,
		IsAccrual:          req.IsAccrual,
		IsRequiredDocument: req.IsRequiredDocument,
	}

	leaveType.ID = id

	if err := h.leaveService.UpdateLeaveType(leaveType, email); err != nil {
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
	_, email, _, ok := getUserContext(c)
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

	if err := h.leaveService.DeleteLeaveType(id, email); err != nil {
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
	userID, email, roles, ok := getUserContext(c)
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
		employee, err := h.employeeService.GetEmployeeByUserID(userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Unable to verify employee authorization",
			})
			return
		}

		if req.EmployeeID != employee.ID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "You can only create leave requests for yourself",
			})
			return
		}
	}

	// Calculate days between start and end date
	days := req.EndDate.Sub(req.StartDate).Hours() / 24

	// Create LeaveRequest entity
	leave := &domain.LeaveRequest{
		EmployeeID:    req.EmployeeID,
		LeaveTypeID:   req.LeaveTypeID,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		RequestedDays: days,
		Reason:        req.Reason,
		Status:        domain.LeaveStatusPending,
		IsPaid:        req.IsPaid,
	}

	if err := h.leaveService.CreateLeave(leave, email, isAdmin(roles)); err != nil {
		status := http.StatusInternalServerError
		// Check for balance validation errors
		if strings.Contains(err.Error(), "insufficient leave balance") ||
			strings.Contains(err.Error(), "no leave balance found") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

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
	userID, email, roles, ok := getUserContext(c)
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

	// Calculate days between start and end date
	days := req.EndDate.Sub(req.StartDate).Hours() / 24

	// Create updated LeaveRequest entity
	leave := &domain.LeaveRequest{
		EmployeeID:    req.EmployeeID,
		LeaveTypeID:   req.LeaveTypeID,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		RequestedDays: days,
		Reason:        req.Reason,
		Status:        existingLeave.Status, // Preserve existing status
		IsPaid:        req.IsPaid,
	}
	leave.ID = id

	if err := h.leaveService.UpdateLeave(leave, email); err != nil {
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

	// Parse sorting parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "created_at"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	result, err := h.leaveService.GetMyLeaveRequestsPaginated(userID, page, limit, sortParams, status)
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

	if !isAdmin(roles) {
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

	// Parse sorting parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "created_at"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	result, err := h.leaveService.GetAllLeaveRequestsPaginated(page, limit, sortParams, status)
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
	userID, email, roles, ok := getUserContext(c)
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

	if err := h.leaveService.ApproveLeave(id, userID, email); err != nil {
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
	userID, email, roles, ok := getUserContext(c)
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

	if err := h.leaveService.RejectLeave(id, req.Reason, email, userID); err != nil {
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
	userID, email, roles, ok := getUserContext(c)
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

	var req CancelLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := h.leaveService.CancelLeave(id, req.Reason, email, userID, isAdmin(roles)); err != nil {
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
