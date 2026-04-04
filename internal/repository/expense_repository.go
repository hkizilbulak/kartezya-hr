package repository

import (
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type ExpenseRepository interface {
	// Expense Request CRUD
	Create(expense *domain.ExpenseRequest) error
	FindByID(id uint) (*domain.ExpenseRequest, error)
	FindByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error)
	FindByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error)
	GetAll(employeeID *uint, page, limit int, sortParams types.SortParams, status string, expenseTypeID *uint, startDate *string, endDate *string) ([]*domain.ExpenseRequest, int64, error)
	Update(expense *domain.ExpenseRequest) error
	Delete(id uint) error
}

type ExpenseTypeRepository interface {
	Create(expenseType *domain.ExpenseType, createdBy string) error
	FindByID(id uint) (*domain.ExpenseType, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.ExpenseType, int64, error)
	GetActive() ([]*domain.ExpenseType, error)
	Update(expenseType *domain.ExpenseType, modifiedBy string) error
	Delete(id uint) error
}

// expenseRepository implements ExpenseRepository
type expenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) ExpenseRepository {
	return &expenseRepository{db: db}
}

func (r *expenseRepository) Create(expense *domain.ExpenseRequest) error {
	return r.db.Create(expense).Error
}

func (r *expenseRepository) FindByID(id uint) (*domain.ExpenseRequest, error) {
	var expense domain.ExpenseRequest
	err := r.db.Preload("Employee").Preload("ExpenseType").Preload("Approver").
		Where("id = ? AND deleted = ?", id, false).First(&expense).Error
	return &expense, err
}

func (r *expenseRepository) FindByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error) {
	var expenses []*domain.ExpenseRequest
	query := r.db.Preload("Employee").Preload("ExpenseType").Preload("Approver").
		Where("employee_id = ? AND deleted = ?", employeeID, false)

	if sortBy != "" {
		orderClause := sortBy
		if sortDir == types.DESC {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	}

	err := query.Find(&expenses).Error
	return expenses, err
}

func (r *expenseRepository) FindByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error) {
	var expenses []*domain.ExpenseRequest
	query := r.db.Preload("Employee").Preload("ExpenseType").Preload("Approver").
		Where("status = ? AND deleted = ?", status, false)

	if sortBy != "" {
		orderClause := sortBy
		if sortDir == types.DESC {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	}

	err := query.Find(&expenses).Error
	return expenses, err
}

func (r *expenseRepository) GetAll(employeeID *uint, page, limit int, sortParams types.SortParams, status string, expenseTypeID *uint, startDate *string, endDate *string) ([]*domain.ExpenseRequest, int64, error) {
	var expenses []*domain.ExpenseRequest
	var total int64

	query := r.db.Preload("Employee").Preload("ExpenseType").Preload("Approver").
		Where("deleted = ?", false)

	// Filter by employee
	if employeeID != nil {
		query = query.Where("employee_id = ?", *employeeID)
	}

	// Filter by status
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if expenseTypeID != nil {
		query = query.Where("expense_type_id = ?", *expenseTypeID)
	}
	if startDate != nil && *startDate != "" {
		query = query.Where("expense_date >= ?", *startDate)
	}
	if endDate != nil && *endDate != "" {
		// Include full day
		query = query.Where("expense_date <= ?", *endDate+" 23:59:59")
	}

	// Count total
	query.Model(&domain.ExpenseRequest{}).Count(&total)

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
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (page - 1) * limit
	err := query.Limit(limit).Offset(offset).Find(&expenses).Error

	return expenses, total, err
}

func (r *expenseRepository) Update(expense *domain.ExpenseRequest) error {
	return r.db.Save(expense).Error
}

func (r *expenseRepository) Delete(id uint) error {
	return r.db.Model(&domain.ExpenseRequest{}).Where("id = ?", id).Update("deleted", true).Error
}

// expenseTypeRepository implements ExpenseTypeRepository
type expenseTypeRepository struct {
	db *gorm.DB
}

func NewExpenseTypeRepository(db *gorm.DB) ExpenseTypeRepository {
	return &expenseTypeRepository{db: db}
}

func (r *expenseTypeRepository) Create(expenseType *domain.ExpenseType, createdBy string) error {
	expenseType.CreatedBy = createdBy
	expenseType.ModifiedBy = createdBy
	return r.db.Create(expenseType).Error
}

func (r *expenseTypeRepository) FindByID(id uint) (*domain.ExpenseType, error) {
	var expenseType domain.ExpenseType
	err := r.db.Where("id = ? AND deleted = ?", id, false).First(&expenseType).Error
	return &expenseType, err
}

func (r *expenseTypeRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.ExpenseType, int64, error) {
	var expenseTypes []*domain.ExpenseType
	var total int64

	query := r.db.Where("deleted = ?", false)

	// Count total
	query.Model(&domain.ExpenseType{}).Count(&total)

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
		query = query.Order("name ASC")
	}

	// Apply pagination if limit > 0
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Find(&expenseTypes).Error
	return expenseTypes, total, err
}

func (r *expenseTypeRepository) GetActive() ([]*domain.ExpenseType, error) {
	var expenseTypes []*domain.ExpenseType
	err := r.db.Where("deleted = ? AND active = ?", false, true).Order("name ASC").Find(&expenseTypes).Error
	return expenseTypes, err
}

func (r *expenseTypeRepository) Update(expenseType *domain.ExpenseType, modifiedBy string) error {
	expenseType.ModifiedBy = modifiedBy
	return r.db.Save(expenseType).Error
}

func (r *expenseTypeRepository) Delete(id uint) error {
	return r.db.Model(&domain.ExpenseType{}).Where("id = ?", id).Update("deleted", true).Error
}
