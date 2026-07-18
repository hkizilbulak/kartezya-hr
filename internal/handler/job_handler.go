package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/jobs"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobService service.JobService
	scheduler  *jobs.Scheduler
}

func NewJobHandler(jobService service.JobService, scheduler *jobs.Scheduler) *JobHandler {
	return &JobHandler{
		jobService: jobService,
		scheduler:  scheduler,
	}
}

func (h *JobHandler) GetJobs(c *gin.Context) {
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	allJobs, err := h.jobService.GetAllJobs(sortParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get jobs"})
		return
	}
	c.JSON(http.StatusOK, allJobs)
}

func (h *JobHandler) GetJobByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := h.jobService.GetJobByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) UpdateJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(uint)

	var updateData struct {
		CronExpression string `json:"cron_expression"`
		IsActive       bool   `json:"is_active"`
		TimeoutSecond  int    `json:"timeout_second"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobUpdate := &domain.Job{
		CronExpression: updateData.CronExpression,
		IsActive:       updateData.IsActive,
		TimeoutSecond:  updateData.TimeoutSecond,
	}

	if err := h.jobService.UpdateJob(uint(id), jobUpdate, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Tell the scheduler to reload this job
	job, _ := h.jobService.GetJobByID(uint(id))
	if job != nil {
		h.scheduler.ReloadJob(job.JobKey)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job updated successfully"})
}

func (h *JobHandler) RunJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := h.jobService.GetJobByID(uint(id))
	if err != nil || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Trigger the job
	err = h.scheduler.TriggerJobNow(job.JobKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job triggered successfully"})
}

func (h *JobHandler) GetJobHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "start_time"),
		Direction: types.NormalizeSortDirection(c.DefaultQuery("direction", "DESC"), "DESC"),
	}

	result, err := h.jobService.GetHistory(uint(id), page, limit, sortParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get job history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Data,
		"page":    result.Page,
	})
}
