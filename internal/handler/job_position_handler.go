package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type JobPositionHandler struct {
	jobPositionService service.JobPositionService
}

func NewJobPositionHandler(jobPositionService service.JobPositionService) *JobPositionHandler {
	return &JobPositionHandler{
		jobPositionService: jobPositionService,
	}
}

// Job Position request/response DTOs
type CreateJobPositionRequest struct {
	Title string `json:"title" binding:"required"`
}

type UpdateJobPositionRequest struct {
	Title string `json:"title" binding:"required"`
}

// CreateJobPosition handles job position creation
// @Summary Create a new job position
// @Description Create a new job position (Admin only)
// @Tags job-positions
// @Accept jsonablo
// @Produce json
// @Security BearerAuth
// @Param request body CreateJobPositionRequest true "Job position data"
// @Success 201 {object} APIResponse{data=domain.JobPosition}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /job-positions [post]
func (h *JobPositionHandler) CreateJobPosition(c *gin.Context) {
	var req CreateJobPositionRequest
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

	jobPosition := &domain.JobPosition{
		Title: req.Title,
	}

	err := h.jobPositionService.CreateJobPosition(jobPosition, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    jobPosition,
		"message": "Job position created successfully",
	})
}

// GetJobPosition handles job position retrieval by ID
// @Summary Get job position by ID
// @Description Get a specific job position by ID (Admin only)
// @Tags job-positions
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job Position ID"
// @Success 200 {object} APIResponse{data=domain.JobPosition}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /job-positions/{id} [get]
func (h *JobPositionHandler) GetJobPosition(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid job position ID",
		})
		return
	}

	jobPosition, err := h.jobPositionService.GetJobPositionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    jobPosition,
	})
}

// GetJobPositions handles paginated job position listing
// @Summary Get job positions with pagination
// @Description Get paginated list of job positions with sorting (Admin only)
// @Tags job-positions
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Success 200 {object} APIResponse{data=[]domain.JobPosition}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /job-positions [get]
func (h *JobPositionHandler) GetJobPositions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	result, err := h.jobPositionService.GetAllJobPositions(page, limit, sortParams)
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

// UpdateJobPosition handles job position updates
// @Summary Update job position
// @Description Update a job position by ID (Admin only)
// @Tags job-positions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job Position ID"
// @Param request body UpdateJobPositionRequest true "Updated job position data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /job-positions/{id} [put]
func (h *JobPositionHandler) UpdateJobPosition(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid job position ID",
		})
		return
	}

	var req UpdateJobPositionRequest
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

	err = h.jobPositionService.UpdateJobPosition(id, req.Title, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job position updated successfully",
	})
}

// DeleteJobPosition handles job position deletion
// @Summary Delete job position
// @Description Delete a job position by ID (Admin only)
// @Tags job-positions
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job Position ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /job-positions/{id} [delete]
func (h *JobPositionHandler) DeleteJobPosition(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid job position ID",
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

	err = h.jobPositionService.DeleteJobPosition(id, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job position deleted successfully",
	})
}
