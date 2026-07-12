package repository

import (
	"fmt"
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
	GetAll(limit, offset int, sortParams types.SortParams, roleID *uint) ([]*domain.ExpenseType, int64, error)
	GetActive(roles []string) ([]*domain.ExpenseType, error)
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

	expTable := domain.GetTableName("hr_expense_requests")

	countQuery := applyExpenseRequestListFilters(r.db.Model(&domain.ExpenseRequest{}), expTable, employeeID, status, expenseTypeID, startDate, endDate)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := applyExpenseRequestListFilters(
		r.db.Model(&domain.ExpenseRequest{}).
			Preload("Employee").
			Preload("ExpenseType").
			Preload("Approver"),
		expTable, employeeID, status, expenseTypeID, startDate, endDate,
	)

	orderClause, needsEmployeeJoin, needsTypeJoin := buildExpenseRequestOrderClause(sortParams.Sort, sortParams.Direction)
	if needsEmployeeJoin {
		empTable := domain.GetTableName("hr_employees")
		query = query.Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.employee_id", empTable, empTable, expTable))
	}
	if needsTypeJoin {
		typeTable := domain.GetTableName("hr_expense_types")
		query = query.Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.expense_type_id", typeTable, typeTable, expTable))
	}
	query = query.Order(orderClause)

	// Apply pagination after ORDER BY
	offset := (page - 1) * limit
	err := query.Limit(limit).Offset(offset).Find(&expenses).Error

	return expenses, total, err
}

// applyExpenseRequestListFilters qualifies main-table columns so employee/type joins
// cannot make shared names (deleted, status, id, …) ambiguous under PostgreSQL.
func applyExpenseRequestListFilters(
	q *gorm.DB,
	expTable string,
	employeeID *uint,
	status string,
	expenseTypeID *uint,
	startDate *string,
	endDate *string,
) *gorm.DB {
	q = q.Where(fmt.Sprintf("%s.deleted = ?", expTable), false)
	if employeeID != nil {
		q = q.Where(fmt.Sprintf("%s.employee_id = ?", expTable), *employeeID)
	}
	if status != "" {
		q = q.Where(fmt.Sprintf("%s.status = ?", expTable), status)
	}
	if expenseTypeID != nil {
		q = q.Where(fmt.Sprintf("%s.expense_type_id = ?", expTable), *expenseTypeID)
	}
	if startDate != nil && *startDate != "" {
		q = q.Where(fmt.Sprintf("%s.expense_date >= ?", expTable), *startDate)
	}
	if endDate != nil && *endDate != "" {
		q = q.Where(fmt.Sprintf("%s.expense_date <= ?", expTable), *endDate+" 23:59:59")
	}
	return q
}

// expenseRequestListFilterExpressions returns the qualified WHERE expressions used by GetAll.
// Tests assert shared columns stay table-qualified for joined sorts.
func expenseRequestListFilterExpressions(expTable string) []string {
	return []string{
		fmt.Sprintf("%s.deleted = ?", expTable),
		fmt.Sprintf("%s.employee_id = ?", expTable),
		fmt.Sprintf("%s.status = ?", expTable),
		fmt.Sprintf("%s.expense_type_id = ?", expTable),
		fmt.Sprintf("%s.expense_date >= ?", expTable),
		fmt.Sprintf("%s.expense_date <= ?", expTable),
	}
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

func (r *expenseTypeRepository) GetAll(limit, offset int, sortParams types.SortParams, roleID *uint) ([]*domain.ExpenseType, int64, error) {
	var expenseTypes []*domain.ExpenseType
	var total int64

	etTable := domain.GetTableName("hr_expense_types")
	query := r.db.Model(&domain.ExpenseType{}).Where(fmt.Sprintf("%s.deleted = ?", etTable), false)

	if roleID != nil {
		query = query.Where(fmt.Sprintf("%s.role_id = ? OR %s.role_id IS NULL", etTable, etTable), *roleID)
	}

	// Count total before optional role join used only for sorting
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderClause, needsRoleJoin := buildExpenseTypeOrderClause(sortParams.Sort, sortParams.Direction)
	if needsRoleJoin {
		rolesTable := domain.GetTableName("hr_roles")
		query = query.Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.role_id", rolesTable, rolesTable, etTable))
	}
	query = query.Order(orderClause)

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Find(&expenseTypes).Error
	return expenseTypes, total, err
}

func (r *expenseTypeRepository) GetActive(roles []string) ([]*domain.ExpenseType, error) {
	var expenseTypes []*domain.ExpenseType

	tableName := domain.ExpenseType{}.TableName()
	rolesTableName := "hr_roles"

	db := r.db.Table(tableName).
		Select(tableName+".*").
		Joins("LEFT JOIN "+rolesTableName+" ON "+rolesTableName+".id = "+tableName+".role_id").
		Where(tableName+".deleted = ? AND "+tableName+".active = ?", false, true)

	if len(roles) > 0 {
		db = db.Where(tableName+".role_id IS NULL OR "+rolesTableName+".name IN ?", roles)
	} else {
		db = db.Where(tableName + ".role_id IS NULL")
	}

	err := db.Order(tableName + ".name ASC").Scan(&expenseTypes).Error
	return expenseTypes, err
}

func (r *expenseTypeRepository) Update(expenseType *domain.ExpenseType, modifiedBy string) error {
	expenseType.ModifiedBy = modifiedBy
	// Use Updates with map to ensure nil/NULL values (like role_id) are persisted correctly.
	// r.db.Save() skips zero-value pointer fields; explicit map forces NULL write.
	return r.db.Model(expenseType).Updates(map[string]interface{}{
		"name":             expenseType.Name,
		"description":      expenseType.Description,
		"requires_receipt": expenseType.RequiresReceipt,
		"max_amount":       expenseType.MaxAmount,
		"active":           expenseType.Active,
		"role_id":          expenseType.RoleID,
		"modified_by":      modifiedBy,
	}).Error
}

func (r *expenseTypeRepository) Delete(id uint) error {
	return r.db.Model(&domain.ExpenseType{}).Where("id = ?", id).Update("deleted", true).Error
}
