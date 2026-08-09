package service

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"

	"github.com/xuri/excelize/v2"
)

type ReportService interface {
	GetWorkDayReport(filter *types.WorkDayReportFilter) (*types.WorkDayReportResponse, error)
	ExportWorkDayReportExcel(request *types.WorkDayReportExportRequest) ([]byte, error)
	GetEforReport(filter *types.WorkDayReportFilter) (*types.EforReportResponse, error)
	GetGradeReportData(filter *types.GradeReportFilter) (*types.GradeReportResponse, error)
	ExportGradeReportExcel(filter *types.GradeReportFilter) ([]byte, error)
	GetContractReportData(filter *types.ContractReportFilter) (*types.ContractReportResponse, error)
	ExportContractReportExcel(request *types.ContractReportExportRequest) ([]byte, error)
}

type reportService struct {
	employeeRepo repository.EmployeeRepository
	workInfoRepo repository.WorkInformationRepository
	leaveRepo    repository.LeaveRepository
	holidayRepo  repository.HolidayRepository
	gradeRepo    repository.GradeRepository
	leaveService LeaveService
	now          func() time.Time
}

func NewReportService(
	employeeRepo repository.EmployeeRepository,
	workInfoRepo repository.WorkInformationRepository,
	leaveRepo repository.LeaveRepository,
	holidayRepo repository.HolidayRepository,
	gradeRepo repository.GradeRepository,
	leaveService LeaveService,
) ReportService {
	return &reportService{
		employeeRepo: employeeRepo,
		workInfoRepo: workInfoRepo,
		leaveRepo:    leaveRepo,
		holidayRepo:  holidayRepo,
		gradeRepo:    gradeRepo,
		leaveService: leaveService,
		now:          time.Now,
	}
}

func (s *reportService) GetWorkDayReport(filter *types.WorkDayReportFilter) (*types.WorkDayReportResponse, error) {
	// Format dates for SQL query
	startDateStr := filter.StartDate.Format("2006-01-02")
	endDateStr := filter.EndDate.Format("2006-01-02")

	// Execute optimized SQL query
	rows, err := s.employeeRepo.GetWorkDayReportData(
		startDateStr,
		endDateStr,
		filter.CompanyID,
		filter.DepartmentIDs,
		filter.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get work day report data: %w", err)
	}

	// Calculate total work days and holiday days once
	totalWorkDays, err := s.leaveService.CalculateWorkingDays(filter.StartDate, filter.EndDate, true, true)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate work days: %w", err)
	}

	totalHolidayDays, err := s.calculateHolidayDays(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate holiday days: %w", err)
	}

	return &types.WorkDayReportResponse{
		StartDate:        filter.StartDate,
		EndDate:          filter.EndDate,
		TotalWorkDays:    totalWorkDays,
		TotalHolidayDays: totalHolidayDays,
		Rows:             rows,
	}, nil
}

func (s *reportService) ExportWorkDayReportExcel(request *types.WorkDayReportExportRequest) ([]byte, error) {
	startDate, err := time.Parse("2006-01-02", request.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", request.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format: %w", err)
	}

	if len(request.ExportColumns) == 0 {
		return nil, fmt.Errorf("export_columns cannot be empty")
	}

	filter := &types.WorkDayReportFilter{
		StartDate:     startDate,
		EndDate:       endDate,
		CompanyID:     request.CompanyID,
		DepartmentIDs: request.DepartmentIDs,
	}

	report, err := s.GetWorkDayReport(filter)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheetName := "Work Day Report"
	f.SetSheetName("Sheet1", sheetName)

	for colIndex, col := range request.ExportColumns {
		headerCell, cellErr := excelize.CoordinatesToCellName(colIndex+1, 1)
		if cellErr != nil {
			return nil, fmt.Errorf("failed to create header cell: %w", cellErr)
		}
		if err = f.SetCellValue(sheetName, headerCell, col.Label); err != nil {
			return nil, fmt.Errorf("failed to write header value: %w", err)
		}
	}

	for rowIndex, row := range report.Rows {
		for colIndex, col := range request.ExportColumns {
			value, valueErr := getFieldValueByJSONKey(row, col.Key)
			if valueErr != nil {
				return nil, valueErr
			}

			dataCell, cellErr := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if cellErr != nil {
				return nil, fmt.Errorf("failed to create data cell: %w", cellErr)
			}

			if err = f.SetCellValue(sheetName, dataCell, value); err != nil {
				return nil, fmt.Errorf("failed to write data value: %w", err)
			}
		}
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel: %w", err)
	}

	return buffer.Bytes(), nil
}

func (s *reportService) GetEforReport(filter *types.WorkDayReportFilter) (*types.EforReportResponse, error) {
	// Get total work day report first
	baseReport, err := s.GetWorkDayReport(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get base work day report: %w", err)
	}

	// Initialize our mapping of employees to ID
	eforMap := make(map[uint]*types.EforReportRow)
	var orderedIDs []uint

	// Map base rows to Efor rows
	for _, row := range baseReport.Rows {
		eforMap[row.ID] = &types.EforReportRow{
			ID:             row.ID,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			IdentityNo:     row.IdentityNo,
			CompanyName:    row.CompanyName,
			DepartmentName: row.DepartmentName,
			Manager:        row.Manager,
			WorkedDays:     0, // Will recalculate total across the range based on new logic
			CurrentGrade:   row.CurrentGrade,
			Grade:          "",
			Rate:           "",
		}
		orderedIDs = append(orderedIDs, row.ID)
	}

	// Calculate current month start for the future month check
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// For each month in the range, get the work day report data and merge
	startMonth := time.Date(filter.StartDate.Year(), filter.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(filter.EndDate.Year(), filter.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	for m := startMonth; !m.After(endMonth); m = m.AddDate(0, 1, 0) {
		// Calculate first and last day of this month
		monthStart := m
		monthEnd := m.AddDate(0, 1, -1)

		// Clamp the month dates to the overall request dates
		queryStart := monthStart
		if queryStart.Before(filter.StartDate) {
			queryStart = filter.StartDate
		}
		queryEnd := monthEnd
		if queryEnd.After(filter.EndDate) {
			queryEnd = filter.EndDate
		}

		monthVal := int(m.Month())

		// If this is the current month or a future month, use system working days for everyone
		if !m.Before(currentMonthStart) {
			// Determine working days from system for this period
			expectedWorkDays, _ := s.leaveService.CalculateWorkingDays(queryStart, queryEnd, true, true)
			totalSystemWorkDays := expectedWorkDays
			if totalSystemWorkDays < 0 {
				totalSystemWorkDays = 0
			}

			for _, eforRow := range eforMap {
				// Base update
				switch monthVal {
				case 1:
					eforRow.January = totalSystemWorkDays
				case 2:
					eforRow.February = totalSystemWorkDays
				case 3:
					eforRow.March = totalSystemWorkDays
				case 4:
					eforRow.April = totalSystemWorkDays
				case 5:
					eforRow.May = totalSystemWorkDays
				case 6:
					eforRow.June = totalSystemWorkDays
				case 7:
					eforRow.July = totalSystemWorkDays
				case 8:
					eforRow.August = totalSystemWorkDays
				case 9:
					eforRow.September = totalSystemWorkDays
				case 10:
					eforRow.October = totalSystemWorkDays
				case 11:
					eforRow.November = totalSystemWorkDays
				case 12:
					eforRow.December = totalSystemWorkDays
				}
				eforRow.WorkedDays += totalSystemWorkDays
			}
		} else {
			// Fetch data for this specific month portion
			monthRows, err := s.employeeRepo.GetWorkDayReportData(
				queryStart.Format("2006-01-02"),
				queryEnd.Format("2006-01-02"),
				filter.CompanyID,
				filter.DepartmentIDs,
				filter.IsActive,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get monthly report data: %w", err)
			}

			// Update our employee mapping with this month's worked days
			for _, row := range monthRows {
				if eforRow, exists := eforMap[row.ID]; exists {
					switch monthVal {
					case 1:
						eforRow.January += row.WorkedDays
					case 2:
						eforRow.February += row.WorkedDays
					case 3:
						eforRow.March += row.WorkedDays
					case 4:
						eforRow.April += row.WorkedDays
					case 5:
						eforRow.May += row.WorkedDays
					case 6:
						eforRow.June += row.WorkedDays
					case 7:
						eforRow.July += row.WorkedDays
					case 8:
						eforRow.August += row.WorkedDays
					case 9:
						eforRow.September += row.WorkedDays
					case 10:
						eforRow.October += row.WorkedDays
					case 11:
						eforRow.November += row.WorkedDays
					case 12:
						eforRow.December += row.WorkedDays
					}
					eforRow.WorkedDays += row.WorkedDays
				}
			}
		}
	}

	// Assemble final row slice
	finalRows := make([]types.EforReportRow, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		finalRows = append(finalRows, *eforMap[id])
	}

	return &types.EforReportResponse{
		StartDate:     baseReport.StartDate,
		EndDate:       baseReport.EndDate,
		TotalWorkDays: baseReport.TotalWorkDays,
		Rows:          finalRows,
	}, nil
}

func (s *reportService) GetGradeReportData(filter *types.GradeReportFilter) (*types.GradeReportResponse, error) {
	rows, err := s.employeeRepo.GetGradeReportData(
		filter.CompanyID,
		filter.DepartmentIDs,
		filter.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get grade report data: %w", err)
	}

	gradeCount, err := s.gradeRepo.GetTotalCount()
	if err != nil {
		return nil, fmt.Errorf("failed to count grades for grade report: %w", err)
	}
	grades, _, err := s.gradeRepo.GetAll(int(gradeCount), 0, types.SortParams{Sort: "id", Direction: "ASC"})
	if err != nil {
		return nil, fmt.Errorf("failed to get grades for grade report: %w", err)
	}
	now := s.now
	if now == nil {
		now = time.Now
	}
	if err := applyExpectedGrades(rows, grades, dateOnlyUTC(now())); err != nil {
		return nil, fmt.Errorf("failed to calculate expected grades: %w", err)
	}

	response := &types.GradeReportResponse{
		Rows: rows,
	}

	return response, nil
}

func applyExpectedGrades(rows []types.GradeReportRow, grades []*domain.Grade, asOfDate time.Time) error {
	sort.SliceStable(grades, func(i, j int) bool {
		left, right := grades[i], grades[j]
		if left.MinYear == nil {
			return false
		}
		if right.MinYear == nil {
			return true
		}
		if *left.MinYear != *right.MinYear {
			return *left.MinYear < *right.MinYear
		}
		return left.ID < right.ID
	})

	gradesByID := make(map[int64]*domain.Grade, len(grades))
	gradesByName := make(map[string]*domain.Grade, len(grades))
	for _, grade := range grades {
		gradesByID[int64(grade.ID)] = grade
		gradesByName[grade.Name] = grade
		gradesByName[normalizeGradeLabel(grade.Name)] = grade
	}

	for index := range rows {
		row := &rows[index]
		row.ExpectedGrade = row.CurrentGrade
		if row.ProfessionStartDate == nil || row.CurrentGrade == "" {
			continue
		}

		currentGrade := resolveCurrentReportGrade(row, gradesByID, gradesByName)
		if currentGrade == nil {
			return fmt.Errorf("employee %d current grade %q could not be resolved", row.ID, row.CurrentGrade)
		}
		if currentGrade.MaxYear == nil {
			continue
		}
		nextGrade := findNextReportGrade(grades, currentGrade)
		if nextGrade == nil {
			continue
		}

		professionStartDate, professionStartErr := parseDate(*row.ProfessionStartDate)
		if professionStartErr != nil || professionStartDate == nil {
			continue
		}

		if expectedGradeTransitionSoon(
			dateOnlyUTC(*professionStartDate),
			asOfDate,
			*currentGrade.MaxYear,
		) {
			row.ExpectedGrade = nextGrade.Name
		}
	}
	return nil
}

func resolveCurrentReportGrade(
	row *types.GradeReportRow,
	gradesByID map[int64]*domain.Grade,
	gradesByName map[string]*domain.Grade,
) *domain.Grade {
	if row.CurrentGradeID != nil {
		if grade := gradesByID[*row.CurrentGradeID]; grade != nil {
			return grade
		}
	}
	if grade := gradesByName[row.CurrentGrade]; grade != nil {
		return grade
	}
	return gradesByName[normalizeGradeLabel(row.CurrentGrade)]
}

func normalizeGradeLabel(label string) string {
	label = strings.TrimSpace(label)
	if rangeStart := strings.Index(label, "("); rangeStart >= 0 {
		label = strings.TrimSpace(label[:rangeStart])
	}
	return strings.ToUpper(label)
}

func findNextReportGrade(grades []*domain.Grade, currentGrade *domain.Grade) *domain.Grade {
	if currentGrade == nil || currentGrade.MaxYear == nil {
		return nil
	}
	for _, grade := range grades {
		if grade.ID != currentGrade.ID && grade.MinYear != nil && *grade.MinYear >= *currentGrade.MaxYear {
			return grade
		}
	}
	return nil
}

func expectedGradeTransitionSoon(professionStartDate, asOfDate time.Time, transitionYear int) bool {
	if transitionYear < 0 || asOfDate.Before(professionStartDate) {
		return false
	}

	nextTransition := addCalendarYears(professionStartDate, transitionYear)
	threshold := addCalendarMonths(nextTransition, -9)
	return !asOfDate.Before(threshold)
}

func addCalendarYears(date time.Time, years int) time.Time {
	return calendarDate(date.Year()+years, date.Month(), date.Day())
}

func addCalendarMonths(date time.Time, months int) time.Time {
	monthIndex := int(date.Month()) - 1 + months
	year := date.Year() + monthIndex/12
	month := monthIndex % 12
	if month < 0 {
		month += 12
		year--
	}
	return calendarDate(year, time.Month(month+1), date.Day())
}

func calendarDate(year int, month time.Month, day int) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// ExportGradeReportExcel exports grade report data as an Excel file
func (s *reportService) ExportGradeReportExcel(filter *types.GradeReportFilter) ([]byte, error) {
	report, err := s.GetGradeReportData(filter)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheetName := "Grade Report"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"Ad", "Soyad", "Şirket", "Departman", "Yönetici",
		"İşe Giriş Tarihi", "Ekip Başlangıç Tarihi", "Meslek Başlangıç Tarihi",
		"Toplam Deneyim", "Mevcut Kademe", "Beklenen Kademe",
	}

	for colIndex, header := range headers {
		cell, cellErr := excelize.CoordinatesToCellName(colIndex+1, 1)
		if cellErr != nil {
			return nil, fmt.Errorf("failed to create header cell: %w", cellErr)
		}
		if err = f.SetCellValue(sheetName, cell, header); err != nil {
			return nil, fmt.Errorf("failed to write header: %w", err)
		}
	}

	for rowIndex, row := range report.Rows {
		rowNum := rowIndex + 2
		values := []interface{}{
			row.FirstName,
			row.LastName,
			row.CompanyName,
			row.DepartmentName,
			row.Manager,
			nilableString(row.HireDate),
			nilableString(row.TeamStartDate),
			nilableString(row.ProfessionStartDate),
			row.TotalExperienceText,
			row.CurrentGrade,
			row.ExpectedGrade,
		}
		for colIndex, val := range values {
			cell, cellErr := excelize.CoordinatesToCellName(colIndex+1, rowNum)
			if cellErr != nil {
				return nil, fmt.Errorf("failed to create data cell: %w", cellErr)
			}
			if err = f.SetCellValue(sheetName, cell, val); err != nil {
				return nil, fmt.Errorf("failed to write data: %w", err)
			}
		}
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel: %w", err)
	}

	return buffer.Bytes(), nil
}

func (s *reportService) GetContractReportData(filter *types.ContractReportFilter) (*types.ContractReportResponse, error) {
	rows, err := s.employeeRepo.GetContractReportData(filter.StartDate, filter.EndDate, filter.CompanyID, filter.DepartmentIDs, filter.IsActive)
	if err != nil {
		return nil, err
	}

	return &types.ContractReportResponse{
		Rows: rows,
	}, nil
}

func (s *reportService) ExportContractReportExcel(request *types.ContractReportExportRequest) ([]byte, error) {
	// 1. Veriyi çek
	filter := &types.ContractReportFilter{
		StartDate:     request.StartDate,
		EndDate:       request.EndDate,
		CompanyID:     request.CompanyID,
		DepartmentIDs: request.DepartmentIDs,
	}

	resp, err := s.GetContractReportData(filter)
	if err != nil {
		return nil, err
	}

	// 2. Excel dosyası oluştur
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheetName := "Sözleşme Raporu"
	f.SetSheetName("Sheet1", sheetName)
	f.DeleteSheet("Sheet1") // Silinmeyebilir eğer adı değiştiyse, ama genel uygulama böyle

	// Başlıkları ayarla
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"4F81BD"},
			Pattern: 1,
		},
	})

	for i, col := range request.ExportColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col.Label)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	for rowIndex, rowData := range resp.Rows {
		v := reflect.ValueOf(rowData)

		for colIndex, colConfig := range request.ExportColumns {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)

			// get Field by exact key mapping (need standard json mapping matching)
			val := ""
			switch colConfig.Key {
			case "fullName":
				val = rowData.FirstName + " " + rowData.LastName
			case "firstName":
				val = rowData.FirstName
			case "lastName":
				val = rowData.LastName
			case "companyName":
				val = rowData.CompanyName
			case "departmentName":
				val = rowData.DepartmentName
			case "manager":
				val = rowData.Manager
			case "contractNames":
				val = rowData.ContractNames
			default:
				// Reflection kullanarak key eşitleme denenebilir,
				// Fakat struct JSON taglari farklı ve Go field isimleri büyük harf.
				field := v.FieldByName(colConfig.Key)
				if field.IsValid() {
					val = fmt.Sprintf("%v", field.Interface())
				}
			}

			f.SetCellValue(sheetName, cell, val)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("could not write excel to buffer: %v", err)
	}

	return buf.Bytes(), nil
}

func getFieldValueByJSONKey(row types.WorkDayReportRow, key string) (interface{}, error) {
	v := reflect.ValueOf(row)
	t := reflect.TypeOf(row)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != key {
			continue
		}

		fieldValue := v.Field(i)
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				return "", nil
			}
			return fieldValue.Elem().Interface(), nil
		}

		return fieldValue.Interface(), nil
	}

	return nil, fmt.Errorf("unsupported export column key: %s", key)
}

// calculateHolidayDays calculates the number of public holiday days in a date range
func (s *reportService) calculateHolidayDays(startDate, endDate time.Time) (float64, error) {
	holidays, err := s.holidayRepo.GetByDateRange(startDate, endDate)
	if err != nil {
		return 0, err
	}

	holidayDays := 0.0
	for _, holiday := range holidays {
		if holiday.IsFullDay {
			holidayDays += 1.0
		} else {
			holidayDays += 0.5
		}
	}

	return holidayDays, nil
}

func nilableString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
