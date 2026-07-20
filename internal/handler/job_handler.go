package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

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

type RunJobRequest struct {
	ReferenceDate string `json:"reference_date"`
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

	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDValue.(uint)

	var req RunJobRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, buildErr := buildJobExecutionContext(req, userID, job.JobKey)
	if buildErr != nil {
		if errors.Is(buildErr, service.ErrPastDateRunNotSupported) {
			c.JSON(http.StatusBadRequest, gin.H{"error": buildErr.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": buildErr.Error()})
		return
	}

	if err := h.jobService.ValidateReferenceDateRun(job.ID, job.JobKey, ctx.ReferenceDate); err != nil {
		if errors.Is(err, service.ErrJobAlreadySucceededForReferenceDate) ||
			errors.Is(err, service.ErrJobAlreadyRunningForReferenceDate) ||
			errors.Is(err, service.ErrJobReferenceDateConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate job run"})
		return
	}

	if err := h.scheduler.TriggerJob(job.JobKey, ctx); err != nil {
		if errors.Is(err, service.ErrJobAlreadySucceededForReferenceDate) ||
			errors.Is(err, service.ErrJobAlreadyRunningForReferenceDate) ||
			errors.Is(err, service.ErrJobReferenceDateConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger job"})
		return
	}

	message := "Job triggered successfully"
	if ctx.ExecutionType == jobs.ExecutionTypeManualBackfill {
		message = "Job triggered successfully for reference date " + ctx.ReferenceDate.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

func buildJobExecutionContext(req RunJobRequest, userID uint, jobKey string) (jobs.JobExecutionContext, error) {
	refDateStr := strings.TrimSpace(req.ReferenceDate)
	if refDateStr == "" {
		return jobs.JobExecutionContext{
			ReferenceDate:     jobs.TodayDate(),
			ExecutionType:     jobs.ExecutionTypeManual,
			TriggeredByUserID: &userID,
		}, nil
	}

	if !jobs.SupportsPastDateRun(jobKey) {
		return jobs.JobExecutionContext{}, service.ErrPastDateRunNotSupported
	}

	refDate, err := jobs.ParseReferenceDate(refDateStr)
	if err != nil {
		return jobs.JobExecutionContext{}, err
	}

	if jobs.IsFutureDate(refDate) {
		return jobs.JobExecutionContext{}, errors.New("reference_date cannot be in the future")
	}

	return jobs.JobExecutionContext{
		ReferenceDate:     refDate,
		ExecutionType:     jobs.ExecutionTypeManualBackfill,
		TriggeredByUserID: &userID,
	}, nil
}

func (h *JobHandler) GetJobHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	history, err := h.jobService.GetHistory(uint(id), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job history"})
		return
	}

	c.JSON(http.StatusOK, history)
}
