package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type EmployeeGradeHandler struct {
	employeeGradeService service.EmployeeGradeService
	employeeService      service.EmployeeService
}

func NewEmployeeGradeHandler(employeeGradeService service.EmployeeGradeService, employeeService service.EmployeeService) *EmployeeGradeHandler {
	return &EmployeeGradeHandler{
		employeeGradeService: employeeGradeService,
		employeeService:      employeeService,
	}
}

type CreateEmployeeGradeRequest struct {
	EmployeeID uint   `json:"employee_id" binding:"required"`
	GradeID    uint   `json:"grade_id" binding:"required"`
	StartDate  string `json:"start_date" binding:"required"`
	EndDate    string `json:"end_date"`
}

type UpdateEmployeeGradeRequest struct {
	EmployeeID uint   `json:"employee_id" binding:"required"`
	GradeID    uint   `json:"grade_id" binding:"required"`
	StartDate  string `json:"start_date" binding:"required"`
	EndDate    string `json:"end_date"`
}

// CreateEmployeeGrade godoc
// @Summary Create employee grade
// @Description Create employee grade record (Admin only)
// @Tags employee-grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateEmployeeGradeRequest true "Employee grade data"
// @Success 201 {object} APIResponse{data=types.EmployeeGradeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-grades [post]
func (h *EmployeeGradeHandler) CreateEmployeeGrade(c *gin.Context) {
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

	var req CreateEmployeeGradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	employeeGrade, err := h.employeeGradeService.CreateEmployeeGrade(req.EmployeeID, req.GradeID, req.StartDate, req.EndDate, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    employeeGrade,
		"message": "Employee grade created successfully",
	})
}

// GetEmployeeGradeByID godoc
// @Summary Get employee grade by ID
// @Description Get specific employee grade by ID
// @Tags employee-grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee Grade ID"
// @Success 200 {object} APIResponse{data=types.EmployeeGradeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /employee-grades/{id} [get]
func (h *EmployeeGradeHandler) GetEmployeeGradeByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee grade ID"})
		return
	}

	employeeGrade, err := h.employeeGradeService.GetEmployeeGradeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, employeeGrade)
}

// GetMyEmployeeGrades godoc
// @Summary Get my employee grades
// @Description Get employee grades for the authenticated employee
// @Tags employee-grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=types.EmployeeGradeWithNames}
// @Failure 401 {object} APIResponse
// @Router /employee-grades/me [get]
func (h *EmployeeGradeHandler) GetMyEmployeeGrades(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	employeeGrades, err := h.employeeGradeService.GetEmployeeGradeByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employeeGrades,
	})
}

// ListEmployeeGrades godoc
// @Summary List employee grades
// @Description Get paginated list of all employee grades (Admin only)
// @Tags employee-grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Param employee_id query int false "Filter by employee ID"
// @Success 200 {object} APIResponse{data=[]types.EmployeeGradeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-grades [get]
func (h *EmployeeGradeHandler) ListEmployeeGrades(c *gin.Context) {
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

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	if sortParams.Direction != "ASC" && sortParams.Direction != "DESC" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sort direction. Must be ASC or DESC"})
		return
	}

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

	response, err := h.employeeGradeService.GetAllEmployeeGrades(page, limit, sortParams, employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateEmployeeGrade godoc
// @Summary Update employee grade
// @Description Update employee grade by ID (Admin only)
// @Tags employee-grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee Grade ID"
// @Param request body UpdateEmployeeGradeRequest true "Updated employee grade data"
// @Success 200 {object} APIResponse{data=types.EmployeeGradeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-grades/{id} [put]
func (h *EmployeeGradeHandler) UpdateEmployeeGrade(c *gin.Context) {
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
			"error":   "Invalid employee grade ID",
		})
		return
	}

	var req UpdateEmployeeGradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := h.employeeGradeService.UpdateEmployeeGrade(id, req.EmployeeID, req.GradeID, req.StartDate, req.EndDate, email, requestingUserID, isAdmin(roles)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	employeeGrade, err := h.employeeGradeService.GetEmployeeGradeByID(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Employee grade updated successfully",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employeeGrade,
		"message": "Employee grade updated successfully",
	})
}

// DeleteEmployeeGrade godoc
// @Summary Delete employee grade
// @Description Delete employee grade by ID (Admin only)
// @Tags employee-grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee Grade ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-grades/{id} [delete]
func (h *EmployeeGradeHandler) DeleteEmployeeGrade(c *gin.Context) {
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
			"error":   "Invalid employee grade ID",
		})
		return
	}

	if err := h.employeeGradeService.DeleteEmployeeGrade(id, email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Employee grade deleted successfully",
	})
}
