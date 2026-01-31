package service

import (
	"fmt"
	"log"
	"sync"
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
	// Calculate working days ONCE (same for all employees)
	commonWorkDays, err := s.leaveService.CalculateWorkingDays(filter.StartDate, filter.EndDate, true, true)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate work days: %w", err)
	}

	// Calculate holiday days ONCE (same for all employees)
	commonHolidayDays, err := s.calculateHolidayDays(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate holiday days: %w", err)
	}

	// Get filtered employees with work info in a single optimized query
	employees, err := s.getFilteredEmployeesWithWorkInfo(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get employees: %w", err)
	}

	// Pre-fetch all leave data for the date range and company/department (if specified)
	leaveDataMap, err := s.getLeaveDataForDateRange(employees, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get leave data: %w", err)
	}

	// Build report rows with concurrent processing
	var rows []types.WorkDayReportRow
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process employees concurrently (max 10 goroutines)
	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)

	for _, employee := range employees {
		wg.Add(1)
		go func(emp *employeeWithWorkInfo) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Get used leave days from pre-fetched data
			usedLeaveDays := leaveDataMap[emp.EmployeeID]

			// Calculate worked days: work days - holiday days - used leave days
			workedDays := commonWorkDays - commonHolidayDays - usedLeaveDays

			row := types.WorkDayReportRow{
				ID:             emp.EmployeeID,
				FirstName:      emp.FirstName,
				LastName:       emp.LastName,
				IdentityNo:     emp.IdentityNo,
				CompanyName:    emp.CompanyName,
				DepartmentName: emp.DepartmentName,
				Manager:        emp.Manager,
				WorkDays:       commonWorkDays,
				HolidayDays:    commonHolidayDays,
				UsedLeaveDays:  usedLeaveDays,
				WorkedDays:     workedDays,
			}

			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()
		}(employee)
	}

	wg.Wait()

	return &types.WorkDayReportResponse{
		StartDate:        filter.StartDate,
		EndDate:          filter.EndDate,
		TotalWorkDays:    commonWorkDays,
		TotalHolidayDays: commonHolidayDays,
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

// getLeaveDataForDateRange pre-fetches all leave data for the date range with company/department filtering
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
