package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	reportService service.ReportService
	emailService  service.EmailService
	config        *config.Config
}

func NewReportHandler(reportService service.ReportService, emailService service.EmailService, cfg *config.Config) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		emailService:  emailService,
		config:        cfg,
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
// @Param department_ids query []int false "Department IDs (supports repeated param or comma separated), example: department_ids=1&department_ids=2 or department_ids=1,2"
// @Param department_id query int false "Legacy single department ID (backward compatibility)"
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
	departmentIDsQuery := c.QueryArray("department_ids")
	legacyDepartmentID := c.Query("department_id")

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

	var departmentIDs []uint
	if len(departmentIDsQuery) == 0 {
		singleValue := c.Query("department_ids")
		if singleValue != "" {
			departmentIDsQuery = append(departmentIDsQuery, singleValue)
		}
	}

	for _, queryValue := range departmentIDsQuery {
		for _, rawID := range strings.Split(queryValue, ",") {
			trimmedID := strings.TrimSpace(rawID)
			if trimmedID == "" {
				continue
			}

			id, parseErr := strconv.ParseUint(trimmedID, 10, 32)
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_ids. Use uint values (comma separated or repeated query params)"})
				return
			}

			departmentIDs = append(departmentIDs, uint(id))
		}
	}

	if legacyDepartmentID != "" {
		id, parseErr := strconv.ParseUint(legacyDepartmentID, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_id"})
			return
		}
		departmentIDs = append(departmentIDs, uint(id))
	}

	// Create filter
	filter := &types.WorkDayReportFilter{
		StartDate:     startDate,
		EndDate:       endDate,
		CompanyID:     companyID,
		DepartmentIDs: departmentIDs,
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActiveBool := isActiveStr == "true"
		filter.IsActive = &isActiveBool
	}

	// Get report
	report, err := h.reportService.GetWorkDayReport(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ExportWorkDayReportExcel godoc
// @Summary Export Work Day Report as Excel
// @Description Export work day report with dynamic export_columns and multiple department_ids
// @Tags reports
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body types.WorkDayReportExportRequest true "Export request"
// @Success 200 {file} binary
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reports/work-day/export [post]
// @Security BearerAuth
func (h *ReportHandler) ExportWorkDayReportExcel(c *gin.Context) {
	var request types.WorkDayReportExportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	excelFile, err := h.reportService.ExportWorkDayReportExcel(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=work-day-report.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelFile)
}

// GetEforReport godoc
// @Summary Get Efor Report
// @Description Get efor report with filters
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string true "Start Date (YYYY-MM-DD)"
// @Param end_date query string true "End Date (YYYY-MM-DD)"
// @Param company_id query uint false "Company ID"
// @Param department_ids query []int false "Department IDs (supports repeated param or comma separated), example: department_ids=1&department_ids=2 or department_ids=1,2"
// @Param department_id query int false "Legacy single department ID (backward compatibility)"
// @Param is_active query bool false "Is Active" default(true)
// @Success 200 {object} types.EforReportResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reports/efor [get]
// @Security BearerAuth
func (h *ReportHandler) GetEforReport(c *gin.Context) {
	// Parse query parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	companyIDStr := c.Query("company_id")
	departmentIDsQuery := c.QueryArray("department_ids")
	legacyDepartmentID := c.Query("department_id")

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

	// Handle department_ids parameter
	var departmentIDs []uint

	// Support comma-separated format
	if len(departmentIDsQuery) == 1 && strings.Contains(departmentIDsQuery[0], ",") {
		parts := strings.Split(departmentIDsQuery[0], ",")
		departmentIDsQuery = parts
	}

	for _, deptIDStr := range departmentIDsQuery {
		deptIDStr = strings.TrimSpace(deptIDStr)
		if deptIDStr != "" {
			id, err := strconv.ParseUint(deptIDStr, 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_ids format"})
				return
			}
			departmentIDs = append(departmentIDs, uint(id))
		}
	}

	// Fallback to legacy single department_id if present and department_ids is empty
	if len(departmentIDs) == 0 && legacyDepartmentID != "" {
		id, err := strconv.ParseUint(legacyDepartmentID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_id"})
			return
		}
		departmentIDs = append(departmentIDs, uint(id))
	}

	var isActive *bool
	isActiveStr := c.DefaultQuery("is_active", "true")
	if isActiveStr != "" {
		active, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid is_active value"})
			return
		}
		isActive = &active
	} else {
		defaultTrue := true
		isActive = &defaultTrue
	}

	filter := &types.WorkDayReportFilter{
		StartDate:     startDate,
		EndDate:       endDate,
		CompanyID:     companyID,
		DepartmentIDs: departmentIDs,
		IsActive:      isActive,
	}

	report, err := h.reportService.GetEforReport(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetGradeReport godoc
// @Summary Get grade distribution report
// @Description Get employee count grouped by grade with optional company and department filters (Admin only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param company_id query int false "Filter by company ID"
// @Param department_id query int false "Filter by department ID"
// @Param department_ids query []int false "Department IDs (supports repeated param or comma separated)"
// @Success 200 {object} types.GradeReportResponse
// @Router /api/reports/grade [get]
func (h *ReportHandler) GetGradeReport(c *gin.Context) {
	companyIDStr := c.Query("company_id")
	departmentIDsQuery := c.QueryArray("department_ids")
	legacyDepartmentID := c.Query("department_id")

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

	var departmentIDs []uint
	if len(departmentIDsQuery) == 1 && strings.Contains(departmentIDsQuery[0], ",") {
		parts := strings.Split(departmentIDsQuery[0], ",")
		departmentIDsQuery = parts
	}

	for _, deptIDStr := range departmentIDsQuery {
		deptIDStr = strings.TrimSpace(deptIDStr)
		if deptIDStr != "" {
			id, err := strconv.ParseUint(deptIDStr, 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_ids format"})
				return
			}
			departmentIDs = append(departmentIDs, uint(id))
		}
	}

	if len(departmentIDs) == 0 && legacyDepartmentID != "" {
		id, err := strconv.ParseUint(legacyDepartmentID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_id"})
			return
		}
		departmentIDs = append(departmentIDs, uint(id))
	}

	filter := &types.GradeReportFilter{
		CompanyID:     companyID,
		DepartmentIDs: departmentIDs,
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActiveBool := isActiveStr == "true"
		filter.IsActive = &isActiveBool
	}

	report, err := h.reportService.GetGradeReportData(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ExportGradeReportExcel godoc
// @Summary Export Grade Report as Excel
// @Description Export grade report data as an Excel file (Admin only)
// @Tags reports
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param request body types.GradeReportExportRequest true "Export request"
// @Success 200 {file} binary
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reports/grade/export [post]
func (h *ReportHandler) ExportGradeReportExcel(c *gin.Context) {
	var req types.GradeReportExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var departmentIDs []uint
	if len(req.DepartmentIDs) > 0 {
		departmentIDs = req.DepartmentIDs
	} else if req.DepartmentID != nil {
		departmentIDs = append(departmentIDs, *req.DepartmentID)
	}

	filter := &types.GradeReportFilter{
		CompanyID:     req.CompanyID,
		DepartmentIDs: departmentIDs,
	}

	excelFile, err := h.reportService.ExportGradeReportExcel(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=grade-report.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelFile)
}

// GetContractReport godoc
// @Summary Get Contract Report
// @Description Get contract report with filters
// @Tags reports
// @Accept json
// @Produce json
// @Param company_id query uint false "Company ID"
// @Param department_ids query []int false "Department IDs (supports repeated param or comma separated)"
// @Param is_active query bool false "Is Active (true or false)"
// @Success 200 {object} types.ContractReportResponse
// @Router /api/reports/contract [get]
func (h *ReportHandler) GetContractReport(c *gin.Context) {
	var filter types.ContractReportFilter

	filter.StartDate = c.Query("start_date")
	filter.EndDate = c.Query("end_date")

	if companyIDStr := c.Query("company_id"); companyIDStr != "" {
		if companyID, err := strconv.ParseUint(companyIDStr, 10, 32); err == nil {
			id := uint(companyID)
			filter.CompanyID = &id
		}
	}

	// Aynı department_ids ayrıştırma mantığı (WorkDayReport'taki gibi)
	var departmentIDs []uint
	if depsStr := c.QueryArray("department_ids"); len(depsStr) > 0 {
		for _, dep := range depsStr {
			parts := strings.Split(dep, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					if id, err := strconv.ParseUint(part, 10, 32); err == nil {
						departmentIDs = append(departmentIDs, uint(id))
					}
				}
			}
		}
	}

	if len(departmentIDs) > 0 {
		filter.DepartmentIDs = departmentIDs
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	report, err := h.reportService.GetContractReportData(&filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ExportContractReportExcel godoc
// @Summary Export Contract Report to Excel
// @Description Creates an Excel file and downloads it
// @Tags reports
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body types.ContractReportExportRequest true "Export Configuration"
// @Success 200 {file} file "sozlesme_raporu.xlsx"
// @Router /api/reports/contract/export/excel [post]
func (h *ReportHandler) ExportContractReportExcel(c *gin.Context) {
	var request types.ContractReportExportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	excelData, err := h.reportService.ExportContractReportExcel(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel file: " + err.Error()})
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=sozlesme_raporu.xlsx")
	c.Data(http.StatusOK, "application/octet-stream", excelData)
}

// SendReportEmail godoc
// @Summary Send report via email
// @Description Send a report file as email attachment to configured recipients
// @Tags reports
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param subject formData string true "Email subject"
// @Param body formData string true "Email body"
// @Param report_type formData string true "Report type"
// @Param recipients formData string true "Recipients JSON array"
// @Param filters formData string false "Filters JSON"
// @Param attachment formData file true "Report file attachment"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /reports/email [post]
func (h *ReportHandler) SendReportEmail(c *gin.Context) {
	subject := c.PostForm("subject")
	body := c.PostForm("body")
	reportType := c.PostForm("report_type")

	if subject == "" || body == "" || reportType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "subject, body and report_type are required",
		})
		return
	}

	// Get recipients from config based on report type
	recipients := h.config.ReportEmail.GetRecipients(reportType)
	if len(recipients) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No recipients configured for report type: " + reportType,
		})
		return
	}

	// Get attachment file
	file, header, err := c.Request.FormFile("attachment")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Attachment file is required",
		})
		return
	}
	defer file.Close()

	log.Printf("[REPORT-EMAIL] Sending report email. Type: %s, Subject: %s, Recipients: %v, File: %s", reportType, subject, recipients, header.Filename)

	// Send email with attachment via Resend
	variables := map[string]interface{}{
		"subject": subject,
		"body":    body,
	}
	err = h.emailService.SendReportEmail(recipients, subject, variables, file, header.Filename)
	if err != nil {
		log.Printf("[REPORT-EMAIL] ERROR: Failed to send report email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send email: " + err.Error(),
		})
		return
	}

	log.Printf("[REPORT-EMAIL] Successfully sent report email to %v", recipients)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Report email sent successfully",
	})
}
