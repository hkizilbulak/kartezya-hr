package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type DepartmentRepository interface {
	Create(department *domain.Department, createdBy string) error
	GetByID(id uint) (*domain.Department, error)
	GetByCompanyIDAndName(companyID uint, name string) (*domain.Department, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Department, int64, error)
	Update(department *domain.Department, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	GetByCompanyID(companyID uint) ([]*domain.Department, error)
}

type departmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) DepartmentRepository {
	return &departmentRepository{db: db}
}

func (r *departmentRepository) Create(department *domain.Department, createdBy string) error {
	department.CreatedBy = createdBy
	department.ModifiedBy = createdBy
	return r.db.Create(department).Error
}

func (r *departmentRepository) GetByID(id uint) (*domain.Department, error) {
	var department domain.Department
	err := r.db.Preload("Company").Preload("EmployeeWorkInformation").
		Where("id = ? AND deleted = ?", id, false).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) GetByCompanyIDAndName(companyID uint, name string) (*domain.Department, error) {
	var department domain.Department
	err := r.db.Where("company_id = ? AND name = ? AND deleted = ?", companyID, name, false).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Department, int64, error) {
	var departments []*domain.Department
	var total int64

	// Validate and sanitize sort field
	validSortFields := map[string]bool{
		"id":         true,
		"company_id": true,
		"name":       true,
		"manager":    true,
		"created_at": true,
		"updated_at": true,
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
	r.db.Model(&domain.Department{}).Where("deleted = ?", false).Count(&total)

	// Get paginated records with sorting
	err := r.db.Preload("Company").Where("deleted = ?", false).
		Order(orderBy).
		Limit(limit).Offset(offset).Find(&departments).Error

	return departments, total, err
}

func (r *departmentRepository) GetByCompanyID(companyID uint) ([]*domain.Department, error) {
	var departments []*domain.Department
	err := r.db.Preload("Company").Where("company_id = ? AND deleted = ?", companyID, false).Find(&departments).Error
	return departments, err
}

func (r *departmentRepository) Update(department *domain.Department, modifiedBy string) error {
	department.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(department).Error
}

func (r *departmentRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.Department{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}
