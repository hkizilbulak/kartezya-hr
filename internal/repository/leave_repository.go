package repository

import (
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type LeaveRepository interface {
	// Leave methods
	Create(leave *domain.LeaveRequest) error
	GetByID(id uint) (*domain.LeaveRequest, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.LeaveRequest, int64, error)
	GetByEmployeeIDWithLeaveType(employeeID uint, limit, offset int, sortParams types.SortParams) ([]*domain.LeaveRequest, int64, error)
	GetByEmployeeIDWithLeaveTypeAndStatus(employeeID uint, limit, offset int, sortParams types.SortParams, status string) ([]*domain.LeaveRequest, int64, error)
	GetAllWithStatus(employeeID *uint, limit, offset int, sortParams types.SortParams, status string, leaveTypeID *uint, startDate *string, endDate *string, listGroup string) ([]*domain.LeaveRequest, int64, error)
	Update(leave *domain.LeaveRequest) error
	Delete(id uint) error
	GetByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetByDateRange(startDate, endDate string) ([]*domain.LeaveRequest, error)
	GetApprovedBirthdayLeaveInYear(employeeID uint, leaveTypeID uint, year int) ([]*domain.LeaveRequest, error)
	GetPendingLeaveByEmployeeAndLeaveType(employeeID uint, leaveTypeID uint) (*domain.LeaveRequest, error)
	GetPendingLeavesByEmployeeIDAndDateRange(employeeID uint, leaveTypeID uint, startDate, endDate time.Time) ([]*domain.LeaveRequest, error)
	GetUsedLeaveDaysByEmployeesInDateRange(employeeIDs []uint, startDate, endDate string) (map[uint]float64, error)
	GetApprovedLeavesByEmployeeAndTypeInYear(employeeID uint, leaveTypeID uint, year int) ([]*domain.LeaveRequest, error)
	GetPendingOrApprovedLeavesByEmployeeAndTypeInYear(employeeID uint, leaveTypeID uint, year int) ([]*domain.LeaveRequest, error)
}

// LeaveTypeRepository interface for leave types
type LeaveTypeRepository interface {
	Create(leaveType *domain.LeaveType, createdBy string) error
	GetByID(id uint) (*domain.LeaveType, error)
	GetByName(name string) (*domain.LeaveType, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.LeaveType, int64, error)
	GetLookup() ([]*domain.LeaveType, error)
	Update(leaveType *domain.LeaveType, modifiedBy string) error
	Delete(id uint) error
}

type leaveRepository struct {
	db *gorm.DB
}

func NewLeaveRepository(db *gorm.DB) LeaveRepository {
	return &leaveRepository{db: db}
}

// Implement Leave methods
func (r *leaveRepository) Create(leave *domain.LeaveRequest) error {
	return r.db.Create(leave).Error
}

func (r *leaveRepository) GetByID(id uint) (*domain.LeaveRequest, error) {
	var leave domain.LeaveRequest
	err := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").First(&leave, id).Error
	if err != nil {
		return nil, err
	}
	return &leave, nil
}

func (r *leaveRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.LeaveRequest, int64, error) {
	var leaves []*domain.LeaveRequest
	var total int64

	// Count total records
	countQuery := r.db.Model(&domain.LeaveRequest{}).Where("deleted = ?", false)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build main query with preloads
	query := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").Where("deleted = ?", false)

	// Apply sorting
	if sortParams.Sort != "" {
		orderClause := sortParams.Sort
		if sortParams.Direction == "DESC" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	} else {
		query = query.Order("id ASC")
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&leaves).Error
	return leaves, total, err
}

func (r *leaveRepository) GetByEmployeeIDWithLeaveType(employeeID uint, limit, offset int, sortParams types.SortParams) ([]*domain.LeaveRequest, int64, error) {
	var leaves []*domain.LeaveRequest
	var total int64

	// Count total records
	countQuery := r.db.Model(&domain.LeaveRequest{}).Where("employee_id = ? AND deleted = ?", employeeID, false)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build main query with preloads
	query := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").Where("employee_id = ? AND deleted = ?", employeeID, false)

	// Apply sorting
	if sortParams.Sort != "" {
		orderClause := sortParams.Sort
		if sortParams.Direction == "DESC" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	} else {
		query = query.Order("id ASC")
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&leaves).Error
	return leaves, total, err
}

func (r *leaveRepository) GetByEmployeeIDWithLeaveTypeAndStatus(employeeID uint, limit, offset int, sortParams types.SortParams, status string) ([]*domain.LeaveRequest, int64, error) {
	var leaves []*domain.LeaveRequest
	var total int64

	// Count total records
	countQuery := r.db.Model(&domain.LeaveRequest{}).Where("employee_id = ? AND status = ? AND deleted = ?", employeeID, status, false)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build main query with preloads
	query := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").Where("employee_id = ? AND status = ? AND deleted = ?", employeeID, status, false)

	// Apply sorting
	if sortParams.Sort != "" {
		orderClause := sortParams.Sort
		if sortParams.Direction == "DESC" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	} else {
		query = query.Order("id ASC")
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&leaves).Error
	return leaves, total, err
}

func (r *leaveRepository) Update(leave *domain.LeaveRequest) error {
	return r.db.Save(leave).Error
}

func (r *leaveRepository) Delete(id uint) error {
	return r.db.Model(&domain.LeaveRequest{}).Where("id = ?", id).Update("deleted", true).Error
}

func (r *leaveRepository) GetByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error) {
	var leaves []*domain.LeaveRequest
	query := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").Where("employee_id = ? AND deleted = ?", employeeID, false)

	if sortBy != "" {
		orderClause := sortBy
		if sortDir == types.DESC {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	}

	err := query.Find(&leaves).Error
	return leaves, err
}

func (r *leaveRepository) GetByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error) {
	var leaves []*domain.LeaveRequest
	query := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").Where("status = ? AND deleted = ?", status, false)

	if sortBy != "" {
		orderClause := sortBy
		if sortDir == types.DESC {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	}

	err := query.Find(&leaves).Error
	return leaves, err
}

func (r *leaveRepository) GetByDateRange(startDate, endDate string) ([]*domain.LeaveRequest, error) {
	var leaves []*domain.LeaveRequest
	err := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").
		Where("start_date >= ? AND end_date <= ? AND deleted = ?", startDate, endDate, false).
		Find(&leaves).Error
	return leaves, err
}

func (r *leaveRepository) GetAllWithStatus(employeeID *uint, limit, offset int, sortParams types.SortParams, status string, leaveTypeID *uint, startDate *string, endDate *string, listGroup string) ([]*domain.LeaveRequest, int64, error) {
	var leaves []*domain.LeaveRequest
	var total int64

	lrTable := domain.GetTableName("hr_leave_requests")

	applyFilters := func(q *gorm.DB) *gorm.DB {
		q = q.Where(fmt.Sprintf("%s.deleted = ?", lrTable), false)
		if clause, args, ok := buildLeaveListGroupClause(listGroup, lrTable); ok {
			q = q.Where(clause, args...)
		}
		if status != "" {
			q = q.Where(fmt.Sprintf("%s.status = ?", lrTable), status)
		}
		if employeeID != nil {
			q = q.Where(fmt.Sprintf("%s.employee_id = ?", lrTable), *employeeID)
		}
		if leaveTypeID != nil {
			q = q.Where(fmt.Sprintf("%s.leave_type_id = ?", lrTable), *leaveTypeID)
		}
		if startDate != nil && *startDate != "" {
			q = q.Where(fmt.Sprintf("%s.start_date >= ?", lrTable), *startDate)
		}
		if endDate != nil && *endDate != "" {
			q = q.Where(fmt.Sprintf("%s.start_date <= ?", lrTable), *endDate+" 23:59:59")
		}
		return q
	}

	countQuery := applyFilters(r.db.Model(&domain.LeaveRequest{}))
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	mainQuery := applyFilters(
		r.db.Model(&domain.LeaveRequest{}).
			Preload("Employee").
			Preload("LeaveType").
			Preload("Approver").
			Preload("Approver.Employee"),
	)

	orderClause, needsEmployeeJoin, needsTypeJoin := buildLeaveRequestOrderClause(sortParams.Sort, sortParams.Direction)
	if needsEmployeeJoin {
		empTable := domain.GetTableName("hr_employees")
		mainQuery = mainQuery.Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.employee_id", empTable, empTable, lrTable))
	}
	if needsTypeJoin {
		typeTable := domain.GetTableName("hr_leave_types")
		mainQuery = mainQuery.Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.leave_type_id", typeTable, typeTable, lrTable))
	}
	mainQuery = mainQuery.Order(orderClause)

	if limit > 0 {
		mainQuery = mainQuery.Limit(limit)
	}
	if offset > 0 {
		mainQuery = mainQuery.Offset(offset)
	}

	err := mainQuery.Find(&leaves).Error
	return leaves, total, err
}

// GetApprovedBirthdayLeaveInYear checks if there's an approved birthday leave in the given year
func (r *leaveRepository) GetApprovedBirthdayLeaveInYear(employeeID uint, leaveTypeID uint, year int) ([]*domain.LeaveRequest, error) {
	var leaves []*domain.LeaveRequest
	// Get all approved birthday leaves for the employee in the given year
	err := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").
		Where("employee_id = ? AND leave_type_id = ? AND status = ? AND EXTRACT(YEAR FROM start_date) = ? AND deleted = ?",
			employeeID, leaveTypeID, "APPROVED", year, false).
		Find(&leaves).Error
	return leaves, err
}

// GetPendingLeaveByEmployeeAndLeaveType checks if there's a pending leave request for the same leave type
func (r *leaveRepository) GetPendingLeaveByEmployeeAndLeaveType(employeeID uint, leaveTypeID uint) (*domain.LeaveRequest, error) {
	var leave domain.LeaveRequest
	err := r.db.Preload("LeaveType").
		Where("employee_id = ? AND leave_type_id = ? AND status = ? AND deleted = ?",
			employeeID, leaveTypeID, "PENDING", false).
		First(&leave).Error

	// Return nil if not found (not an error for our use case)
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &leave, nil
}

// GetPendingLeavesByEmployeeIDAndDateRange checks if there are pending or approved leave requests
// for the same leave type that overlap with the given date range
// A leave overlaps if: leave.start_date <= endDate AND leave.end_date >= startDate
func (r *leaveRepository) GetPendingLeavesByEmployeeIDAndDateRange(employeeID uint, leaveTypeID uint, startDate, endDate time.Time) ([]*domain.LeaveRequest, error) {
	var leaves []*domain.LeaveRequest
	err := r.db.Preload("Employee").Preload("LeaveType").Preload("Approver").Preload("Approver.Employee").
		Where("employee_id = ? AND leave_type_id = ? AND status IN (?, ?) AND deleted = ?",
			employeeID, leaveTypeID, "PENDING", "APPROVED", false).
		Where("start_date <= ? AND end_date >= ?", endDate, startDate).
		Find(&leaves).Error
	return leaves, err
}

// Leave Type Repository implementation
type leaveTypeRepository struct {
	db *gorm.DB
}

func NewLeaveTypeRepository(db *gorm.DB) LeaveTypeRepository {
	return &leaveTypeRepository{db: db}
}

func (r *leaveTypeRepository) Create(leaveType *domain.LeaveType, createdBy string) error {
	leaveType.CreatedBy = createdBy
	leaveType.ModifiedBy = createdBy
	return r.db.Create(leaveType).Error
}

func (r *leaveTypeRepository) GetByID(id uint) (*domain.LeaveType, error) {
	var leaveType domain.LeaveType
	err := r.db.First(&leaveType, id).Error
	if err != nil {
		return nil, err
	}
	return &leaveType, nil
}

func (r *leaveTypeRepository) GetByName(name string) (*domain.LeaveType, error) {
	var leaveType domain.LeaveType
	err := r.db.Where("name = ? AND deleted = ?", name, false).First(&leaveType).Error
	if err != nil {
		return nil, err
	}
	return &leaveType, nil
}

func (r *leaveTypeRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.LeaveType, int64, error) {
	var leaveTypes []*domain.LeaveType
	var total int64

	// Count total records
	countQuery := r.db.Model(&domain.LeaveType{}).Where("deleted = ?", false)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build main query — allowlisted ORDER BY before LIMIT/OFFSET
	query := r.db.Where("deleted = ?", false).
		Order(buildLeaveTypeOrderClause(sortParams.Sort, sortParams.Direction))

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&leaveTypes).Error
	return leaveTypes, total, err
}

func (r *leaveTypeRepository) GetLookup() ([]*domain.LeaveType, error) {
	var leaveTypes []*domain.LeaveType
	err := r.db.Select("id, name, description, limit_amount, is_required_document").Where("deleted = ?", false).Order("id ASC").Find(&leaveTypes).Error
	return leaveTypes, err
}

func (r *leaveTypeRepository) Update(leaveType *domain.LeaveType, modifiedBy string) error {
	leaveType.ModifiedBy = modifiedBy
	return r.db.Save(leaveType).Error
}

func (r *leaveTypeRepository) Delete(id uint) error {
	return r.db.Model(&domain.LeaveType{}).Where("id = ?", id).Update("deleted", true).Error
}

// GetUsedLeaveDaysByEmployeesInDateRange returns a map of employee_id -> total used leave days
// for approved leaves that overlap with the given date range
func (r *leaveRepository) GetUsedLeaveDaysByEmployeesInDateRange(employeeIDs []uint, startDate, endDate string) (map[uint]float64, error) {
	type Result struct {
		EmployeeID uint
		TotalDays  float64
	}

	var results []Result

	// Query to sum up requested_days for APPROVED leaves that overlap with date range
	// A leave overlaps if: leave.start_date <= endDate AND leave.end_date >= startDate
	err := r.db.Model(&domain.LeaveRequest{}).
		Select("employee_id, SUM(requested_days) as total_days").
		Where("employee_id IN ?", employeeIDs).
		Where("status = ?", "APPROVED").
		Where("deleted = ?", false).
		Where("start_date <= ?", endDate).
		Where("end_date >= ?", startDate).
		Group("employee_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert to map
	usedDaysMap := make(map[uint]float64)
	for _, result := range results {
		usedDaysMap[result.EmployeeID] = result.TotalDays
	}

	return usedDaysMap, nil
}

// GetApprovedLeavesByEmployeeAndTypeInYear returns all approved leave requests for an employee
// of a specific leave type within a given year
func (r *leaveRepository) GetApprovedLeavesByEmployeeAndTypeInYear(employeeID uint, leaveTypeID uint, year int) ([]*domain.LeaveRequest, error) {
	var leaves []*domain.LeaveRequest

	// Calculate start and end of the year
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.UTC)

	err := r.db.
		Where("employee_id = ?", employeeID).
		Where("leave_type_id = ?", leaveTypeID).
		Where("status = ?", "APPROVED").
		Where("deleted = ?", false).
		Where("start_date >= ? AND start_date <= ?", yearStart, yearEnd).
		Order("start_date DESC").
		Find(&leaves).Error

	if err != nil {
		return nil, err
	}

	return leaves, nil
}

// GetPendingOrApprovedLeavesByEmployeeAndTypeInYear returns all pending or approved leave requests for an employee
// of a specific leave type within a given year
func (r *leaveRepository) GetPendingOrApprovedLeavesByEmployeeAndTypeInYear(employeeID uint, leaveTypeID uint, year int) ([]*domain.LeaveRequest, error) {
	var leaves []*domain.LeaveRequest

	// Calculate start and end of the year
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 31, 23, 59, 59, 999999999, time.UTC)

	err := r.db.
		Where("employee_id = ?", employeeID).
		Where("leave_type_id = ?", leaveTypeID).
		Where("status IN (?, ?)", "PENDING", "APPROVED").
		Where("deleted = ?", false).
		Where("start_date >= ? AND start_date <= ?", yearStart, yearEnd).
		Order("start_date DESC").
		Find(&leaves).Error

	if err != nil {
		return nil, err
	}

	return leaves, nil
}
