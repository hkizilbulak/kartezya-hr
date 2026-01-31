package handler

import (
	"net/http"
	"strconv"
	"time"

	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	reportService service.ReportService
}

func NewReportHandler(reportService service.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// GetWorkDayReport godoc
// @Summary Get Work Day Report
// @Description Get work day report with filters
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string true "Start Date (YYYY-MM-DD)"
// @Param end_date query string true "End Date (YYYY-MM-DD)"
// @Param company_id query uint false "Company ID"
// @Param department_id query uint false "Department ID"
// @Param is_active query bool false "Is Active" default(true)
// @Success 200 {object} types.WorkDayReportResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reports/work-day [get]
// @Security BearerAuth
func (h *ReportHandler) GetWorkDayReport(c *gin.Context) {
	// Parse query parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	companyIDStr := c.Query("company_id")
	departmentIDStr := c.Query("department_id")
	isActiveStr := c.DefaultQuery("is_active", "true")

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use YYYY-MM-DD"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use YYYY-MM-DD"})
		return
	}

	// Parse optional parameters
	var companyID *uint
	if companyIDStr != "" {
		id, err := strconv.ParseUint(companyIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company_id"})
			return
		}
		uid := uint(id)
		companyID = &uid
	}

	var departmentID *uint
	if departmentIDStr != "" {
		id, err := strconv.ParseUint(departmentIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_id"})
			return
		}
		uid := uint(id)
		departmentID = &uid
	}

	// Parse isActive flag
	isActive := isActiveStr == "true" || isActiveStr == "1"

	// Create filter
	filter := &types.WorkDayReportFilter{
		StartDate:    startDate,
		EndDate:      endDate,
		CompanyID:    companyID,
		DepartmentID: departmentID,
		IsActive:     isActive,
	}

	// Get report
	report, err := h.reportService.GetWorkDayReport(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}
