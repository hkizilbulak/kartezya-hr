package service

import (
	"fmt"
	"reflect"
	"time"

	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"

	"github.com/xuri/excelize/v2"
)

type ReportService interface {
	GetWorkDayReport(filter *types.WorkDayReportFilter) (*types.WorkDayReportResponse, error)
	ExportWorkDayReportExcel(request *types.WorkDayReportExportRequest) ([]byte, error)
	GetGradeReportData(filter *types.GradeReportFilter) (*types.GradeReportResponse, error)
	ExportGradeReportExcel(filter *types.GradeReportFilter) ([]byte, error)
}

type reportService struct {
	employeeRepo repository.EmployeeRepository
	workInfoRepo repository.WorkInformationRepository
	leaveRepo    repository.LeaveRepository
	holidayRepo  repository.HolidayRepository
	leaveService LeaveService
}

func NewReportService(
	employeeRepo repository.EmployeeRepository,
	workInfoRepo repository.WorkInformationRepository,
	leaveRepo repository.LeaveRepository,
	holidayRepo repository.HolidayRepository,
	leaveService LeaveService,
) ReportService {
	return &reportService{
		employeeRepo: employeeRepo,
		workInfoRepo: workInfoRepo,
		leaveRepo:    leaveRepo,
		holidayRepo:  holidayRepo,
		leaveService: leaveService,
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

// GetGradeReportData returns grade report data for a given company and department
func (s *reportService) GetGradeReportData(filter *types.GradeReportFilter) (*types.GradeReportResponse, error) {
	rows, err := s.employeeRepo.GetGradeReportData(
		filter.CompanyID,
		filter.DepartmentID,
		filter.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get grade report data: %w", err)
	}

	response := &types.GradeReportResponse{
		Rows: rows,
	}

	return response, nil
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
		"Toplam Boşluk (Yıl)", "Toplam Deneyim", "Mevcut Kademe", "Beklenen Kademe",
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
			row.TotalGap,
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

func nilableString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
