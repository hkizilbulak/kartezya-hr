package service

import (
	"fmt"
	"log"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type ReportService interface {
	GetWorkDayReport(filter *types.WorkDayReportFilter) (*types.WorkDayReportResponse, error)
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

// Helper struct to hold employee with work info
type employeeWithWorkInfo struct {
	EmployeeID     uint
	FirstName      string
	LastName       string
	IdentityNo     string
	CompanyID      uint
	DepartmentID   uint
	CompanyName    string
	DepartmentName string
	Manager        string
}

// getFilteredEmployeesWithWorkInfo gets employees with their work info in a single optimized query
func (s *reportService) getFilteredEmployeesWithWorkInfo(filter *types.WorkDayReportFilter) ([]*employeeWithWorkInfo, error) {
	// Get all work information records with related data
	workInfos, _, err := s.workInfoRepo.GetAll(10000, 0, types.SortParams{Sort: "id", Direction: "ASC"}, nil)
	if err != nil {
		return nil, err
	}

	// Get all employees
	employees, _, err := s.employeeRepo.GetAll(10000, 0, types.SortParams{Sort: "id", Direction: "ASC"})
	if err != nil {
		return nil, err
	}

	// Create a map of work info by employee ID for quick lookup
	workInfoMap := make(map[uint]*domain.EmployeeWorkInformation)
	for i := range workInfos {
		workInfoMap[workInfos[i].EmployeeID] = &workInfos[i]
	}

	// Filter and combine employees with their work info
	var result []*employeeWithWorkInfo

	for _, emp := range employees {
		if emp == nil {
			continue
		}

		// Filter by active/deleted status
		if filter.IsActive && emp.Deleted {
			continue
		}

		// Get work info for this employee
		workInfo, exists := workInfoMap[emp.ID]
		if !exists {
			continue
		}

		// Filter by company and department if specified
		if filter.CompanyID != nil && workInfo.CompanyID != *filter.CompanyID {
			continue
		}

		if filter.DepartmentID != nil && workInfo.DepartmentID != *filter.DepartmentID {
			continue
		}

		// Get company and department names (should be pre-loaded)
		companyName := workInfo.Company.Name
		departmentName := workInfo.Department.Name
		manager := workInfo.Department.Manager

		empWithInfo := &employeeWithWorkInfo{
			EmployeeID:     emp.ID,
			FirstName:      emp.FirstName,
			LastName:       emp.LastName,
			IdentityNo:     emp.IdentityNo,
			CompanyID:      workInfo.CompanyID,
			DepartmentID:   workInfo.DepartmentID,
			CompanyName:    companyName,
			DepartmentName: departmentName,
			Manager:        manager,
		}

		result = append(result, empWithInfo)
	}

	return result, nil
}

// getLeaveBalanceData fetches used leave days for the specific date range using optimized SQL
func (s *reportService) getLeaveBalanceData(employees []*employeeWithWorkInfo, filter *types.WorkDayReportFilter) (map[uint]float64, error) {
	// Initialize map with 0 for all employees
	leaveBalanceMap := make(map[uint]float64)
	employeeIDs := make([]uint, 0, len(employees))

	for _, emp := range employees {
		leaveBalanceMap[emp.EmployeeID] = 0
		employeeIDs = append(employeeIDs, emp.EmployeeID)
	}

	if len(employeeIDs) == 0 {
		return leaveBalanceMap, nil
	}

	log.Printf("[DEBUG] getLeaveBalanceData: Fetching approved leaves for %d employees in date range: %s to %s",
		len(employees), filter.StartDate.Format("2006-01-02"), filter.EndDate.Format("2006-01-02"))

	// Get used leave days from database using optimized aggregate query
	usedDaysMap, err := s.leaveRepo.GetUsedLeaveDaysByEmployeesInDateRange(
		employeeIDs,
		filter.StartDate.Format("2006-01-02"),
		filter.EndDate.Format("2006-01-02"),
	)
	if err != nil {
		log.Printf("[ERROR] getLeaveBalanceData: Failed to fetch used leave days: %v", err)
		return leaveBalanceMap, nil // Return initialized map on error (non-blocking)
	}

	// Merge results
	for employeeID, usedDays := range usedDaysMap {
		leaveBalanceMap[employeeID] = usedDays
		log.Printf("[DEBUG] getLeaveBalanceData: Employee %d has %.1f used leave days in date range", employeeID, usedDays)
	}

	log.Printf("[DEBUG] getLeaveBalanceData: Completed. Processed %d employees", len(employees))

	return leaveBalanceMap, nil
}

// getLeaveDataForDateRange is deprecated - leave balance is now managed in leave_balance table
// This method is kept for backward compatibility but should not be used
// DEPRECATED: Use getLeaveBalanceData instead
func (s *reportService) getLeaveDataForDateRange(employees []*employeeWithWorkInfo, filter *types.WorkDayReportFilter) (map[uint]float64, error) {
	leaveDataMap := make(map[uint]float64)

	// Initialize all employees with 0 leave days
	employeeIDMap := make(map[uint]bool)
	for _, emp := range employees {
		leaveDataMap[emp.EmployeeID] = 0
		employeeIDMap[emp.EmployeeID] = true
	}

	log.Printf("[DEBUG] getLeaveDataForDateRange: Starting with %d employees, date range: %s to %s",
		len(employees), filter.StartDate.Format("2006-01-02"), filter.EndDate.Format("2006-01-02"))

	// Get all approved leave requests with APPROVED status
	log.Printf("[DEBUG] getLeaveDataForDateRange: Fetching approved leave requests...")
	leaves, _, err := s.leaveRepo.GetAllWithStatus(10000, 0, types.SortParams{Sort: "id", Direction: "ASC"}, "APPROVED")
	if err != nil {
		log.Printf("[ERROR] getLeaveDataForDateRange: Failed to fetch leave requests: %v", err)
		return leaveDataMap, nil // Return empty map on error (non-blocking)
	}
	log.Printf("[DEBUG] getLeaveDataForDateRange: Retrieved %d approved leave requests", len(leaves))

	// Pre-load holidays for the date range to exclude them from leave calculations
	log.Printf("[DEBUG] getLeaveDataForDateRange: Fetching holidays for date range...")
	holidays, err := s.holidayRepo.GetByDateRange(filter.StartDate, filter.EndDate)
	if err != nil {
		log.Printf("[WARN] getLeaveDataForDateRange: Failed to fetch holidays: %v", err)
		holidays = make([]*domain.Holiday, 0)
	}
	log.Printf("[DEBUG] getLeaveDataForDateRange: Retrieved %d holidays", len(holidays))

	// Create holiday date map for quick lookup
	holidayMap := make(map[string]bool)
	for _, holiday := range holidays {
		if holiday != nil {
			holidayMap[holiday.HolidayDate.Format("2006-01-02")] = true
		}
	}

	// Calculate used leave days per employee
	processedLeaves := 0
	skippedLeaves := 0
	for i := range leaves {
		leave := leaves[i]

		// Status already filtered by GetAllWithStatus, but double-check
		if leave.Status != "APPROVED" {
			skippedLeaves++
			continue
		}

		// Only count if employee is in our filtered list
		if !employeeIDMap[leave.EmployeeID] {
			skippedLeaves++
			continue
		}

		// Check if leave overlaps with the date range
		// Leave overlaps if: leave.StartDate <= filter.EndDate AND leave.EndDate >= filter.StartDate
		if leave.StartDate.After(filter.EndDate) || leave.EndDate.Before(filter.StartDate) {
			log.Printf("[DEBUG] getLeaveDataForDateRange: Leave ID %d (Employee %d) doesn't overlap with date range. Leave: %s to %s",
				leave.ID, leave.EmployeeID, leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))
			skippedLeaves++
			continue
		}

		// Calculate overlapping days
		overlapStart := leave.StartDate
		if overlapStart.Before(filter.StartDate) {
			overlapStart = filter.StartDate
		}

		overlapEnd := leave.EndDate
		if overlapEnd.After(filter.EndDate) {
			overlapEnd = filter.EndDate
		}

		log.Printf("[DEBUG] getLeaveDataForDateRange: Processing Leave ID %d (Employee %d), overlap: %s to %s",
			leave.ID, leave.EmployeeID, overlapStart.Format("2006-01-02"), overlapEnd.Format("2006-01-02"))

		// Count the overlapping working days (exclude weekends and public holidays)
		daysInRange := 0.0
		currentDate := overlapStart
		for {
			dayOfWeek := currentDate.Weekday()
			isWeekend := dayOfWeek == time.Saturday || dayOfWeek == time.Sunday
			isHoliday := holidayMap[currentDate.Format("2006-01-02")]

			// Only count working days (not weekend and not public holiday)
			if !isWeekend && !isHoliday {
				daysInRange += 1.0
			}

			if currentDate.Equal(overlapEnd) {
				break
			}

			currentDate = currentDate.AddDate(0, 0, 1)
		}

		log.Printf("[DEBUG] getLeaveDataForDateRange: Leave ID %d (Employee %d) counted as %.1f days",
			leave.ID, leave.EmployeeID, daysInRange)
		leaveDataMap[leave.EmployeeID] += daysInRange
		processedLeaves++
	}

	log.Printf("[DEBUG] getLeaveDataForDateRange: Completed. Processed %d leaves, Skipped %d leaves", processedLeaves, skippedLeaves)

	return leaveDataMap, nil
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
