package jobs

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ContractStatusInfoJob generates a weekly contract status report and sends it via email.
type ContractStatusInfoJob struct {
	db                *gorm.DB
	emailService      service.EmailService
	mailConfigService service.MailConfigService
}

func NewContractStatusInfoJob(db *gorm.DB, emailService service.EmailService, mailConfigService service.MailConfigService) *ContractStatusInfoJob {
	return &ContractStatusInfoJob{
		db:                db,
		emailService:      emailService,
		mailConfigService: mailConfigService,
	}
}

// contractStatusRow holds one row from the DB query
type contractStatusRow struct {
	ID                   uint
	ContractNo           string
	ProjectName          string
	CustomerContactName  string
	CustomerContactEmail string
	Status               string
	StartDate            time.Time
	EndDate              *time.Time
	EmployeeCount        int
}

// Run fetches contracts by target statuses, builds an Excel, and sends the report email.
func (j *ContractStatusInfoJob) Run(_ JobExecutionContext) (int, error) {
	log.Println("[ContractStatusInfoJob] Starting contract status report job...")

	targetStatuses := []string{
		domain.ContractStatusPendingProposal,
		domain.ContractStatusProposalSent,
		domain.ContractStatusPendingApproval,
		domain.ContractStatusPendingRevision,
		domain.ContractStatusProposalRevision,
		domain.ContractStatusApproved,
	}

	// ── 1. Fetch contracts ──────────────────────────────────────────────────
	var contracts []domain.Contract
	if err := j.db.
		Where("status IN ? AND deleted = ?", targetStatuses, false).
		Order("status ASC, start_date DESC").
		Find(&contracts).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch contracts: %w", err)
	}

	if len(contracts) == 0 {
		log.Println("[ContractStatusInfoJob] No contracts found, skipping email.")
		return 0, nil
	}

	// ── 2. Count by status ─────────────────────────────────────────────────
	statusLabels := map[string]string{
		domain.ContractStatusPendingProposal:  "Teklif Aşamasında",
		domain.ContractStatusProposalSent:     "Teklif İletildi",
		domain.ContractStatusProposalRevision: "Teklif Revizyon",
		domain.ContractStatusPendingRevision:  "Revizyon Bekleniyor",
		domain.ContractStatusPendingApproval:  "Onay Bekleniyor",
		domain.ContractStatusApproved:         "Onaylandı",
	}

	countByStatus := make(map[string]int)
	for _, c := range contracts {
		countByStatus[c.Status]++
	}

	// ── 3. Build Excel ─────────────────────────────────────────────────────
	excelBytes, err := j.buildExcel(contracts, statusLabels)
	if err != nil {
		return 0, fmt.Errorf("failed to build excel: %w", err)
	}

	// ── 4. Build HTML mail body ────────────────────────────────────────────
	now := time.Now()
	body := fmt.Sprintf(
		"<p>Merhaba <strong>%s</strong> tarihli haftalık sözleşme durum raporu ektedir.</p>"+
			"<table border='1' cellpadding='6' cellspacing='0' style='border-collapse:collapse;'>"+
			"<tr><th>Durum</th><th>Adet</th></tr>",
		now.Format("02.01.2006"),
	)

	orderedStatuses := []string{
		domain.ContractStatusPendingProposal,
		domain.ContractStatusProposalSent,
		domain.ContractStatusProposalRevision,
		domain.ContractStatusPendingRevision,
		domain.ContractStatusPendingApproval,
		domain.ContractStatusApproved,
	}
	for _, st := range orderedStatuses {
		if cnt, ok := countByStatus[st]; ok {
			body += fmt.Sprintf("<tr><td>%s</td><td>%d</td></tr>", statusLabels[st], cnt)
		}
	}
	body += fmt.Sprintf("<tr><td><strong>Toplam</strong></td><td><strong>%d</strong></td></tr>", len(contracts))
	body += "</table>"

	// ── 5. Resolve recipients from REPORT_EMAIL_CONTRACT_STATUS config ─────
	toList, ccList, bccList, templateCode, cfgErr := j.mailConfigService.ResolveRecipients("REPORT_EMAIL_CONTRACT_STATUS")
	if cfgErr != nil || len(toList) == 0 {
		log.Printf("[ContractStatusInfoJob] REPORT_EMAIL_CONTRACT_STATUS config not found or empty, skipping email: %v", cfgErr)
		return len(contracts), nil
	}

	if templateCode == "" {
		templateCode = "report-mail" // fallback template
	}

	subject := fmt.Sprintf("Haftalık Sözleşme Durum Raporu — %s", now.Format("02.01.2006"))
	variables := map[string]interface{}{
		"body": body,
	}

	// ── 6. Send email with Excel attachment ────────────────────────────────
	filename := fmt.Sprintf("sozlesme_raporu_%s.xlsx", now.Format("20060102"))
	if err := j.emailService.SendReportEmail(
		toList, ccList, bccList,
		subject,
		variables,
		bytes.NewReader(excelBytes),
		filename,
	); err != nil {
		return len(contracts), fmt.Errorf("failed to send contract report email: %w", err)
	}

	log.Printf("[ContractStatusInfoJob] Contract report sent to %v. Total contracts: %d", toList, len(contracts))
	return len(contracts), nil
}

// buildExcel creates an Excel workbook with one sheet per status group.
func (j *ContractStatusInfoJob) buildExcel(contracts []domain.Contract, statusLabels map[string]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Group by status
	grouped := make(map[string][]domain.Contract)
	for _, c := range contracts {
		grouped[c.Status] = append(grouped[c.Status], c)
	}

	orderedStatuses := []string{
		domain.ContractStatusPendingProposal,
		domain.ContractStatusProposalSent,
		domain.ContractStatusProposalRevision,
		domain.ContractStatusPendingRevision,
		domain.ContractStatusPendingApproval,
		domain.ContractStatusApproved,
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4F81BD"}, Pattern: 1},
	})

	firstSheet := true
	for _, status := range orderedStatuses {
		rows, ok := grouped[status]
		if !ok {
			continue
		}

		label := statusLabels[status]
		if label == "" {
			label = status
		}

		sheetName := label
		if len(sheetName) > 31 { // Excel sheet name limit
			sheetName = sheetName[:31]
		}

		if firstSheet {
			f.SetSheetName("Sheet1", sheetName)
			firstSheet = false
		} else {
			f.NewSheet(sheetName)
		}

		headers := []string{"Sözleşme No", "Proje Adı", "Müşteri", "Müşteri E-Posta", "Durum", "Başlangıç Tarihi", "Bitiş Tarihi"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		for ri, c := range rows {
			endDateStr := "-"
			if c.EndDate != nil {
				endDateStr = c.EndDate.Format("02.01.2006")
			}

			rowVals := []interface{}{
				c.ContractNo,
				c.ProjectName,
				c.CustomerContactName,
				c.CustomerContactEmail,
				statusLabels[c.Status],
				c.StartDate.Format("02.01.2006"),
				endDateStr,
			}
			for ci, val := range rowVals {
				cell, _ := excelize.CoordinatesToCellName(ci+1, ri+2)
				f.SetCellValue(sheetName, cell, val)
			}
		}

		// Auto-fit columns (approximate)
		for i := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheetName, col, col, 22)
		}
	}

	// Add summary sheet
	summarySheet := "Özet"
	f.NewSheet(summarySheet)
	f.SetCellValue(summarySheet, "A1", "Durum")
	f.SetCellValue(summarySheet, "B1", "Adet")
	f.SetCellStyle(summarySheet, "A1", "B1", headerStyle)

	row := 2
	total := 0
	for _, status := range orderedStatuses {
		if rows, ok := grouped[status]; ok {
			cell1, _ := excelize.CoordinatesToCellName(1, row)
			cell2, _ := excelize.CoordinatesToCellName(2, row)
			f.SetCellValue(summarySheet, cell1, statusLabels[status])
			f.SetCellValue(summarySheet, cell2, len(rows))
			total += len(rows)
			row++
		}
	}
	totalCell1, _ := excelize.CoordinatesToCellName(1, row)
	totalCell2, _ := excelize.CoordinatesToCellName(2, row)
	f.SetCellValue(summarySheet, totalCell1, "Toplam")
	f.SetCellValue(summarySheet, totalCell2, total)

	// Move summary to first position
	f.SetSheetVisible(summarySheet, true)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
