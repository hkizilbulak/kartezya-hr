package handler

import (
"net/http"
"strconv"

"kartezya-hr/internal/domain"
"kartezya-hr/internal/service"
"kartezya-hr/internal/types"

"github.com/gin-gonic/gin"
)

type GradeHandler struct {
	gradeService service.GradeService
}

func NewGradeHandler(gradeService service.GradeService) *GradeHandler {
	return &GradeHandler{
		gradeService: gradeService,
	}
}

// Grade request/response DTOs
type CreateGradeRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateGradeRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateGrade godoc
// @Summary Create grade
// @Description Create a new grade (Admin only)
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param grade body CreateGradeRequest true "Grade data"
// @Success 201 {object} APIResponse{data=types.GradeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /grades [post]
func (h *GradeHandler) CreateGrade(c *gin.Context) {
	var req CreateGradeRequest
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

	grade := &domain.Grade{
		Name:        req.Name,
		Description: req.Description,
	}

	err := h.gradeService.CreateGrade(grade, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
"success": false,
"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
"success": true,
"data":    grade,
"message": "Grade created successfully",
})
}

// GetGrade godoc
// @Summary Get grade by ID
// @Description Get a specific grade by ID (Admin only)
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Grade ID"
// @Success 200 {object} APIResponse{data=types.GradeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /grades/{id} [get]
func (h *GradeHandler) GetGrade(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
"success": false,
"error":   "Invalid grade ID",
})
		return
	}

	grade, err := h.gradeService.GetGradeByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
"success": false,
"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
"success": true,
"data":    grade,
})
}

// GetGrades godoc
// @Summary Get grades with pagination
// @Description Get paginated list of grades with sorting (Admin only)
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Success 200 {object} APIResponse{data=[]types.GradeResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /grades [get]
func (h *GradeHandler) GetGrades(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	result, err := h.gradeService.GetAllGrades(page, limit, sortParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
"success": false,
"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
"data":    result.Data,
"page":    result.Page,
"success": true,
})
}

// UpdateGrade godoc
// @Summary Update grade
// @Description Update a grade by ID (Admin only)
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Grade ID"
// @Param request body UpdateGradeRequest true "Updated grade data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /grades/{id} [put]
func (h *GradeHandler) UpdateGrade(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
"success": false,
"error":   "Invalid grade ID",
})
		return
	}

	var req UpdateGradeRequest
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

	grade := &domain.Grade{
		Name:        req.Name,
		Description: req.Description,
	}

	err = h.gradeService.UpdateGrade(id, grade, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
"success": false,
"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
"success": true,
"message": "Grade updated successfully",
})
}

// DeleteGrade godoc
// @Summary Delete grade
// @Description Delete a grade by ID (Admin only)
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Grade ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /grades/{id} [delete]
func (h *GradeHandler) DeleteGrade(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
"success": false,
"error":   "Invalid grade ID",
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

	err = h.gradeService.DeleteGrade(id, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
"success": false,
"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
"success": true,
"message": "Grade deleted successfully",
})
}
