package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type EmployeeRepository interface {
	Create(employee *domain.Employee, createdBy string) error
	GetByID(id uint) (*domain.Employee, error)
	GetByUserID(userID uint) (*domain.Employee, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Employee, int64, error)
	Update(employee *domain.Employee, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type employeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) Create(employee *domain.Employee, createdBy string) error {
	employee.CreatedBy = createdBy
	employee.ModifiedBy = createdBy
	return r.db.Create(employee).Error
}

func (r *employeeRepository) GetByID(id uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Preload("User").Where("deleted = ?", false).First(&employee, id).Error
	return &employee, err
}

func (r *employeeRepository) GetByUserID(userID uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Preload("User").Where("user_id = ? AND deleted = ?", userID, false).First(&employee).Error
	return &employee, err
}

func (r *employeeRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Employee, int64, error) {
	var employees []*domain.Employee
	var total int64

	// Validate and sanitize sort field
	validSortFields := map[string]bool{
		"id":            true,
		"user_id":       true,
		"employee_id":   true,
		"first_name":    true,
		"last_name":     true,
		"phone":         true,
		"address":       true,
		"date_of_birth": true,
		"hire_date":     true,
		"created_at":    true,
		"updated_at":    true,
	}

	sortField := "id"
	if validSortFields[sortParams.Sort] {
		sortField = sortParams.Sort
	}

	direction := "ASC"
	if sortParams.Direction == "DESC" {
		direction = "DESC"
	}

	orderBy := fmt.Sprintf("%s %s", sortField, direction)

	// Count total records
	r.db.Model(&domain.Employee{}).Where("deleted = ?", false).Count(&total)

	// Get paginated records with sorting
	err := r.db.Preload("User").
		Where("deleted = ?", false).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&employees).Error

	return employees, total, err
}

func (r *employeeRepository) Update(employee *domain.Employee, modifiedBy string) error {
	employee.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(employee).Error
}

func (r *employeeRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.Employee{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}
