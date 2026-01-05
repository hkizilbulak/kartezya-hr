package repository

import (
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type LeaveBalanceRepository interface {
	Create(leaveBalance *domain.LeaveBalance, createdBy string) error
	GetByID(id uint) (*domain.LeaveBalance, error)
	GetByEmployeeAndLeaveType(employeeID, leaveTypeID uint) ([]*domain.LeaveBalance, error)
	GetByEmployeeAndLeaveTypeForYear(employeeID, leaveTypeID uint, year int) (*domain.LeaveBalance, error)
	GetByEmployeeIDPaginated(employeeID uint, limit, offset int, sortParams types.SortParams) ([]*domain.LeaveBalance, int64, error)
	Update(leaveBalance *domain.LeaveBalance, modifiedBy string) error
	UpdateMultiple(leaveBalances []*domain.LeaveBalance, modifiedBy string) error
	Delete(id uint) error
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.LeaveBalance, int64, error)
}

type leaveBalanceRepository struct {
	db *gorm.DB
}

func NewLeaveBalanceRepository(db *gorm.DB) LeaveBalanceRepository {
	return &leaveBalanceRepository{db: db}
}

func (r *leaveBalanceRepository) Create(leaveBalance *domain.LeaveBalance, createdBy string) error {
	leaveBalance.CreatedBy = createdBy
	leaveBalance.ModifiedBy = createdBy
	return r.db.Create(leaveBalance).Error
}

func (r *leaveBalanceRepository) GetByID(id uint) (*domain.LeaveBalance, error) {
	var leaveBalance domain.LeaveBalance
	err := r.db.Preload("Employee").Preload("LeaveType").
		Where("id = ? AND deleted = ?", id, false).
		First(&leaveBalance).Error
	return &leaveBalance, err
}

func (r *leaveBalanceRepository) GetByEmployeeAndLeaveType(employeeID, leaveTypeID uint) ([]*domain.LeaveBalance, error) {
	var leaveBalances []*domain.LeaveBalance
	err := r.db.Where("employee_id = ? AND leave_type_id = ? AND deleted = ?", employeeID, leaveTypeID, false).
		Order("year ASC").
		Find(&leaveBalances).Error
	return leaveBalances, err
}

func (r *leaveBalanceRepository) GetByEmployeeAndLeaveTypeForYear(employeeID, leaveTypeID uint, year int) (*domain.LeaveBalance, error) {
	var leaveBalance domain.LeaveBalance
	err := r.db.Where("employee_id = ? AND leave_type_id = ? AND year = ? AND deleted = ?", employeeID, leaveTypeID, year, false).
		First(&leaveBalance).Error
	return &leaveBalance, err
}

func (r *leaveBalanceRepository) GetByEmployeeIDPaginated(employeeID uint, limit, offset int, sortParams types.SortParams) ([]*domain.LeaveBalance, int64, error) {
	var leaveBalances []*domain.LeaveBalance
	var total int64

	query := r.db.Model(&domain.LeaveBalance{}).Where("employee_id = ? AND deleted = ?", employeeID, false)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := sortParams.Sort + " " + string(sortParams.Direction)
	query = query.Order(orderClause)

	// Apply pagination
	if err := query.Limit(limit).Offset(offset).Preload("Employee").Preload("LeaveType").Find(&leaveBalances).Error; err != nil {
		return nil, 0, err
	}

	return leaveBalances, total, nil
}

func (r *leaveBalanceRepository) Update(leaveBalance *domain.LeaveBalance, modifiedBy string) error {
	leaveBalance.ModifiedBy = modifiedBy
	return r.db.Save(leaveBalance).Error
}

func (r *leaveBalanceRepository) UpdateMultiple(leaveBalances []*domain.LeaveBalance, modifiedBy string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, balance := range leaveBalances {
			balance.ModifiedBy = modifiedBy
			if err := tx.Save(balance).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *leaveBalanceRepository) Delete(id uint) error {
	return r.db.Model(&domain.LeaveBalance{}).Where("id = ?", id).Update("deleted", true).Error
}

func (r *leaveBalanceRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.LeaveBalance, int64, error) {
	var leaveBalances []*domain.LeaveBalance
	var total int64

	query := r.db.Model(&domain.LeaveBalance{}).Where("deleted = ?", false)

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := sortParams.Sort + " " + string(sortParams.Direction)
	query = query.Order(orderClause)

	// Apply pagination
	if err := query.Limit(limit).Offset(offset).Preload("Employee").Preload("LeaveType").Find(&leaveBalances).Error; err != nil {
		return nil, 0, err
	}

	return leaveBalances, total, nil
}
