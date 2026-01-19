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
	GetTotalCount() (int64, error)
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

	// Determine sort field and direction
	sortField := "d.id"
	direction := "ASC"

	if sortParams.Direction == "DESC" {
		direction = "DESC"
	}

	// Map frontend sort fields to database columns
	if sortParams.Sort != "" {
		switch sortParams.Sort {
		case "company":
			sortField = "c.name"
		case "name":
			sortField = "d.name"
		case "manager":
			sortField = "d.manager"
		case "created_at":
			sortField = "d.created_at"
		case "updated_at":
			sortField = "d.updated_at"
		default:
			sortField = "d.id"
		}
	}

	// Build SQL query with proper JOIN
	query := `
		SELECT d.* FROM hr_departments d
		LEFT JOIN hr_companies c ON c.id = d.company_id
		WHERE d.deleted = false
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`
	formattedQuery := fmt.Sprintf(query, sortField, direction)

	// Execute query
	err := r.db.Raw(formattedQuery, limit, offset).
		Preload("Company").
		Find(&departments).Error

	if err != nil {
		return nil, 0, err
	}

	// Count total records
	r.db.Model(&domain.Department{}).Where("deleted = ?", false).Count(&total)

	return departments, total, nil
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

// GetTotalCount returns the total number of departments
func (r *departmentRepository) GetTotalCount() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Department{}).Where("deleted = ?", false).Count(&count).Error
	return count, err
}
