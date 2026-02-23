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
		Where(fmt.Sprintf("%s.id = ? AND %s.deleted = ?", domain.GetTableName("hr_departments"), domain.GetTableName("hr_departments")), id, false).
		First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) GetByCompanyIDAndName(companyID uint, name string) (*domain.Department, error) {
	var department domain.Department
	err := r.db.Where(fmt.Sprintf("%s.company_id = ? AND %s.name = ? AND %s.deleted = ?",
		domain.GetTableName("hr_departments"),
		domain.GetTableName("hr_departments"),
		domain.GetTableName("hr_departments")), companyID, name, false).
		First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Department, int64, error) {
	var departments []*domain.Department
	var total int64

	// Get dynamic table names
	deptTable := domain.GetTableName("hr_departments")
	companyTable := domain.GetTableName("hr_companies")

	// Determine sort field and direction
	sortField := fmt.Sprintf("%s.id", deptTable)
	direction := "ASC"

	if sortParams.Direction == "DESC" {
		direction = "DESC"
	}

	// Map frontend sort fields to database columns
	if sortParams.Sort != "" {
		switch sortParams.Sort {
		case "company":
			sortField = fmt.Sprintf("%s.name", companyTable)
		case "name":
			sortField = fmt.Sprintf("%s.name", deptTable)
		case "manager":
			sortField = fmt.Sprintf("%s.manager", deptTable)
		case "created_at":
			sortField = fmt.Sprintf("%s.created_at", deptTable)
		case "updated_at":
			sortField = fmt.Sprintf("%s.updated_at", deptTable)
		default:
			sortField = fmt.Sprintf("%s.id", deptTable)
		}
	}

	// Build SQL query with proper JOIN
	query := fmt.Sprintf(`
		SELECT %s.* FROM %s
		LEFT JOIN %s ON %s.id = %s.company_id
		WHERE %s.deleted = false
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, deptTable, deptTable, companyTable, companyTable, deptTable, deptTable, sortField, direction)

	// Execute query
	err := r.db.Raw(query, limit, offset).
		Preload("Company").
		Find(&departments).Error

	if err != nil {
		return nil, 0, err
	}

	// Count total records
	r.db.Model(&domain.Department{}).Where(fmt.Sprintf("%s.deleted = ?", deptTable), false).Count(&total)

	return departments, total, nil
}

func (r *departmentRepository) GetByCompanyID(companyID uint) ([]*domain.Department, error) {
	var departments []*domain.Department
	err := r.db.Preload("Company").
		Where(fmt.Sprintf("%s.company_id = ? AND %s.deleted = ?",
			domain.GetTableName("hr_departments"),
			domain.GetTableName("hr_departments")), companyID, false).
		Find(&departments).Error
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
	err := r.db.Model(&domain.Department{}).
		Where(fmt.Sprintf("%s.deleted = ?", domain.GetTableName("hr_departments")), false).
		Count(&count).Error
	return count, err
}
