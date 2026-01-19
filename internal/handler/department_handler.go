package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	departmentService service.DepartmentService
}

func NewDepartmentHandler(departmentService service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{
		departmentService: departmentService,
	}
}

// Department request/response DTOs
type CreateDepartmentRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Manager   string `json:"manager"`
}

type UpdateDepartmentRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Manager   string `json:"manager"`
}

// CreateDepartment handles department creation
// @Summary Create a new department
// @Description Create a new department (Admin only)
// @Tags departments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateDepartmentRequest true "Department data"
// @Success 201 {object} APIResponse{data=types.DepartmentResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /departments [post]
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get current user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	department := &domain.Department{
		CompanyID: req.CompanyID,
		Name:      req.Name,
		Manager:   req.Manager,
	}

	err := h.departmentService.CreateDepartment(department, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    department,
		"message": "Department created successfully",
	})
}

// GetDepartment handles department retrieval by ID
// @Summary Get department by ID
// @Description Get a specific department by ID (Admin only)
// @Tags departments
// @Produce json
// @Security BearerAuth
// @Param id path int true "Department ID"
// @Success 200 {object} APIResponse{data=types.DepartmentResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /departments/{id} [get]
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid department ID",
		})
		return
	}

	department, err := h.departmentService.GetDepartmentByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    department,
	})
}

// GetDepartments handles department listing with pagination
// @Summary List departments
// @Description Get all departments with pagination (Admin only)
// @Tags departments
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Param company_id query int false "Filter by company ID"
// @Param sort query string false "Sort by field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Success 200 {object} APIResponse{data=service.PaginatedResponse}
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /departments [get]
func (h *DepartmentHandler) GetDepartments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	result, err := h.departmentService.GetAllDepartments(page, limit, sortParams)
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

// UpdateDepartment handles department updates
// @Summary Update department
// @Description Update a department by ID (Admin only)
// @Tags departments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Department ID"
// @Param request body UpdateDepartmentRequest true "Updated department data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /departments/{id} [put]
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid department ID",
		})
		return
	}

	var req UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get current user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	department := &domain.Department{
		CompanyID: req.CompanyID,
		Name:      req.Name,
		Manager:   req.Manager,
	}

	err = h.departmentService.UpdateDepartment(id, department, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Department updated successfully",
	})
}

// DeleteDepartment handles department deletion
// @Summary Delete department
// @Description Delete a department by ID (Admin only)
// @Tags departments
// @Produce json
// @Security BearerAuth
// @Param id path int true "Department ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /departments/{id} [delete]
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid department ID",
		})
		return
	}

	// Get current user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	err = h.departmentService.DeleteDepartment(id, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Department deleted successfully",
	})
}
