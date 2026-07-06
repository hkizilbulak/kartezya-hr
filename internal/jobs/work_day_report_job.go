package jobs

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/xuri/excelize/v2"
)

// WorkDayReportJob generates a monthly work-day report for the previous month and sends it via email.
type WorkDayReportJob struct {
	reportService     service.ReportService
	emailService      service.EmailService
	mailConfigService service.MailConfigService
}

func NewWorkDayReportJob(
	reportService service.ReportService,
	emailService service.EmailService,
	mailConfigService service.MailConfigService,
) *WorkDayReportJob {
	return &WorkDayReportJob{
		reportService:     reportService,
		emailService:      emailService,
		mailConfigService: mailConfigService,
	}
}

// Run generates the previous month's work-day report and sends it via REPORT_EMAIL_WORK_DAY.
func (j *WorkDayReportJob) Run() (int, error) {
	log.Println("[WorkDayReportJob] Starting work day report job...")

	// ── 1. Previous month date range ───────────────────────────────────────
	now := time.Now()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	startDate := firstOfThisMonth.AddDate(0, -1, 0) // first day of prev month
	endDate := firstOfThisMonth.AddDate(0, 0, -1)   // last day of prev month

	monthLabel := startDate.Format("01.2006") // e.g. "06.2026"

	log.Printf("[WorkDayReportJob] Reporting period: %s — %s",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// ── 2. Fetch report data ────────────────────────────────────────────────
	isActive := true
	filter := &types.WorkDayReportFilter{
		StartDate: startDate,
		EndDate:   endDate,
		IsActive:  &isActive,
	}

	report, err := j.reportService.GetWorkDayReport(filter)
	if err != nil {
		return 0, fmt.Errorf("failed to get work day report: %w", err)
	}

	if len(report.Rows) == 0 {
		log.Println("[WorkDayReportJob] No rows in report, skipping email.")
		return 0, nil
	}

	// ── 3. Calculate totals ─────────────────────────────────────────────────
	var totalUsedLeave float64
	for _, row := range report.Rows {
		totalUsedLeave += row.UsedLeaveDays
	}

	// ── 4. Build Excel ──────────────────────────────────────────────────────
	excelBytes, err := j.buildExcel(report, monthLabel, totalUsedLeave)
	if err != nil {
		return 0, fmt.Errorf("failed to build excel: %w", err)
	}

	// ── 5. Build HTML body ──────────────────────────────────────────────────
	body := fmt.Sprintf(
		"<p>Merhaba,</p>"+
			"<p><strong>%s</strong> dönemine ait aylık çalışma günü raporu ektedir.</p>"+
			"<table border='1' cellpadding='6' cellspacing='0' style='border-collapse:collapse;'>"+
			"<tr><th>Toplam İş Günü</th><th>Toplam Kullanılan İzin</th><th>Çalışan Sayısı</th></tr>"+
			"<tr><td>%.1f</td><td>%.1f</td><td>%d</td></tr>"+
			"</table>",
		monthLabel,
		report.TotalWorkDays,
		totalUsedLeave,
		len(report.Rows),
	)

	// ── 6. Resolve recipients ───────────────────────────────────────────────
	toList, ccList, bccList, templateCode, cfgErr := j.mailConfigService.ResolveRecipients("REPORT_EMAIL_WORK_DAY")
	if cfgErr != nil || len(toList) == 0 {
		log.Printf("[WorkDayReportJob] REPORT_EMAIL_WORK_DAY config not found or empty, skipping email: %v", cfgErr)
		return len(report.Rows), nil
	}

	if templateCode == "" {
		templateCode = "report-mail"
	}

	subject := fmt.Sprintf("Aylık Çalışma Günü Raporu — %s", monthLabel)
	variables := map[string]interface{}{
		"subject": subject,
		"body":    body,
	}

	// ── 6. Send email ───────────────────────────────────────────────────────
	filename := fmt.Sprintf("calisma_gunu_raporu_%s.xlsx", startDate.Format("200601"))
	if err := j.emailService.SendReportEmail(
		toList, ccList, bccList,
		subject,
		variables,
		bytes.NewReader(excelBytes),
		filename,
	); err != nil {
		return len(report.Rows), fmt.Errorf("failed to send work day report email: %w", err)
	}

	log.Printf("[WorkDayReportJob] Work day report sent to %v. Period: %s, Rows: %d",
		toList, monthLabel, len(report.Rows))
	return len(report.Rows), nil
}

// buildExcel creates an Excel workbook with the work day report data.
func (j *WorkDayReportJob) buildExcel(report *types.WorkDayReportResponse, monthLabel string, totalUsedLeave float64) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Çalışma Günü Raporu"
	f.SetSheetName("Sheet1", sheet)

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4F81BD"}, Pattern: 1},
	})

	headers := []string{
		"AD SOYAD", "İŞ GÜNÜ", "KULLANILAN İZİN", "ÇALIŞILAN GÜN",
		"ŞİRKET", "DEPARTMAN", "YÖNETİCİ",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	for ri, row := range report.Rows {
		rowNum := ri + 2
		vals := []interface{}{
			row.FirstName + " " + row.LastName,
			row.WorkDays,
			row.UsedLeaveDays,
			row.WorkedDays,
			row.CompanyName,
			row.DepartmentName,
			row.Manager,
		}
		for ci, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(ci+1, rowNum)
			f.SetCellValue(sheet, cell, v)
		}
	}

	// Column widths
	colWidths := []float64{28, 12, 16, 14, 22, 22, 22}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	// Summary sheet
	sumSheet := "Özet"
	f.NewSheet(sumSheet)

	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4F81BD"}, Pattern: 1},
	})

	summaryHeaders := []string{"Dönem", "Toplam İş Günü", "Toplam Kullanılan İzin", "Çalışan Sayısı"}
	for i, h := range summaryHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sumSheet, cell, h)
		f.SetCellStyle(sumSheet, cell, cell, summaryStyle)
	}

	summaryVals := []interface{}{
		monthLabel,
		report.TotalWorkDays,
		totalUsedLeave,
		len(report.Rows),
	}
	for i, v := range summaryVals {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sumSheet, cell, v)
	}

	for i, w := range []float64{12, 16, 14, 16} {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sumSheet, col, col, w)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
