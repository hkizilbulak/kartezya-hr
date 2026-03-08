package service

import (
	"fmt"
	"time"

	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type ReportService interface {
	GetWorkDayReport(filter *types.WorkDayReportFilter) (*types.WorkDayReportResponse, error)
	GetGradeReportData(filter *types.GradeReportFilter) (*types.GradeReportResponse, error)
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
		filter.DepartmentID,
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
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get grade report data: %w", err)
	}

	response := &types.GradeReportResponse{
		Rows: rows,
	}

	return response, nil
}
