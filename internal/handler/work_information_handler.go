package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type WorkInformationHandler struct {
	workInfoService service.WorkInformationService
	employeeService service.EmployeeService
}

func NewWorkInformationHandler(workInfoService service.WorkInformationService, employeeService service.EmployeeService) *WorkInformationHandler {
	return &WorkInformationHandler{
		workInfoService: workInfoService,
		employeeService: employeeService,
	}
}

type CreateWorkInformationRequest struct {
	EmployeeID    uint   `json:"employee_id" binding:"required"`
	CompanyID     uint   `json:"company_id" binding:"required"`
	DepartmentID  uint   `json:"department_id" binding:"required"`
	JobPositionID uint   `json:"job_position_id" binding:"required"`
	StartDate     string `json:"start_date" binding:"required"`
	EndDate       string `json:"end_date"`
	PersonnelNo   string `json:"personnel_no"`
	WorkEmail     string `json:"work_email"`
}

type UpdateWorkInformationRequest struct {
	EmployeeID    uint   `json:"employee_id" binding:"required"`
	CompanyID     uint   `json:"company_id" binding:"required"`
	DepartmentID  uint   `json:"department_id" binding:"required"`
	JobPositionID uint   `json:"job_position_id" binding:"required"`
	StartDate     string `json:"start_date" binding:"required"`
	EndDate       string `json:"end_date"`
	PersonnelNo   string `json:"personnel_no"`
	WorkEmail     string `json:"work_email"`
}

// CreateWorkInformation godoc
// @Summary Create work information
// @Description Create work information for an employee (Admin only)
// @Tags work-information
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateWorkInformationRequest true "Work information data"
// @Success 201 {object} APIResponse{data=types.WorkInformationResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /work-information [post]
func (h *WorkInformationHandler) CreateWorkInformation(c *gin.Context) {
	_, email, roles, ok := getUserContext(c)
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
			"error":   "Admin access required",
		})
		return
	}

	var req CreateWorkInformationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	workInfo, err := h.workInfoService.CreateWorkInformation(req.EmployeeID, req.CompanyID, req.DepartmentID, req.JobPositionID, req.StartDate, req.EndDate, req.PersonnelNo, req.WorkEmail, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    workInfo,
		"message": "Work information created successfully",
	})
}

// GetWorkInformationByID godoc
// @Summary Get work information by ID
// @Description Get specific work information by ID
// @Tags work-information
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work Information ID"
// @Success 200 {object} APIResponse{data=types.WorkInformationResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /work-information/{id} [get]
func (h *WorkInformationHandler) GetWorkInformationByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work information ID"})
		return
	}

	workInfo, err := h.workInfoService.GetWorkInformationByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Service zaten istenen formatta (WorkInformationResponse) döndürüyor
	c.JSON(http.StatusOK, workInfo)
}

// GetMyWorkInformation godoc
// @Summary Get my work information
// @Description Get work information for the authenticated employee
// @Tags work-information
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=types.WorkInformationWithNames}
// @Failure 401 {object} APIResponse
// @Router /work-information/me [get]
func (h *WorkInformationHandler) GetMyWorkInformation(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	workInfos, err := h.workInfoService.GetWorkInformationByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workInfos,
	})
}

// ListWorkInformation godoc
// @Summary List work information
// @Description Get paginated list of all work information (Admin only)
// @Tags work-information
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Param employee_id query int false "Filter by employee ID"
// @Success 200 {object} APIResponse{data=[]types.WorkInformationResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /work-information [get]
func (h *WorkInformationHandler) ListWorkInformation(c *gin.Context) {
	// Parse page and limit parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	// Parse sort parameters
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	// Validate sort direction
	if sortParams.Direction != "ASC" && sortParams.Direction != "DESC" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sort direction. Must be ASC or DESC"})
		return
	}

	// Parse employee_id filter parameter
	var employeeID *uint
	if employeeIDStr := c.Query("employee_id"); employeeIDStr != "" {
		id, err := strconv.ParseUint(employeeIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee_id parameter"})
			return
		}
		empID := uint(id)
		employeeID = &empID
	}

	// Get work information list from service
	response, err := h.workInfoService.GetAllWorkInformations(page, limit, sortParams, employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateWorkInformation godoc
// @Summary Update work information
// @Description Update work information by ID (Admin only)
// @Tags work-information
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work Information ID"
// @Param request body UpdateWorkInformationRequest true "Updated work information data"
// @Success 200 {object} APIResponse{data=types.WorkInformationResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /work-information/{id} [put]
func (h *WorkInformationHandler) UpdateWorkInformation(c *gin.Context) {
	requestingUserID, email, roles, ok := getUserContext(c)
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
			"error":   "Admin access required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid work information ID",
		})
		return
	}

	var req UpdateWorkInformationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := h.workInfoService.UpdateWorkInformation(id, req.EmployeeID, req.CompanyID, req.DepartmentID, req.JobPositionID, req.StartDate, req.EndDate, req.PersonnelNo, req.WorkEmail, email, requestingUserID, isAdmin(roles)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Güncellenmiş work information'ı döndür
	workInfo, err := h.workInfoService.GetWorkInformationByID(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Work information updated successfully",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workInfo,
		"message": "Work information updated successfully",
	})
}

// DeleteWorkInformation godoc
// @Summary Delete work information
// @Description Delete work information by ID (Admin only)
// @Tags work-information
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Work Information ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /work-information/{id} [delete]
func (h *WorkInformationHandler) DeleteWorkInformation(c *gin.Context) {
	_, email, roles, ok := getUserContext(c)
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
			"error":   "Admin access required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid work information ID",
		})
		return
	}

	if err := h.workInfoService.DeleteWorkInformation(id, email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Work information deleted successfully",
	})
}
