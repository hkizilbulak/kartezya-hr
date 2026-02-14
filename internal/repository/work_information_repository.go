package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type WorkInformationRepository interface {
	Create(workInfo *domain.EmployeeWorkInformation, createdBy string) error
	GetByID(id uint) (*domain.EmployeeWorkInformation, error)
	GetByEmployeeID(employeeID uint) ([]domain.EmployeeWorkInformation, error)
	GetByUserID(userID uint) ([]domain.EmployeeWorkInformation, error)
	GetAll(limit, offset int, sortParams types.SortParams, employeeID *uint) ([]domain.EmployeeWorkInformation, int64, error)
	Update(workInfo *domain.EmployeeWorkInformation, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type workInformationRepository struct {
	db *gorm.DB
}

func NewWorkInformationRepository(db *gorm.DB) WorkInformationRepository {
	return &workInformationRepository{
		db: db,
	}
}

func (r *workInformationRepository) Create(workInfo *domain.EmployeeWorkInformation, createdBy string) error {
	workInfo.CreatedBy = createdBy
	workInfo.ModifiedBy = createdBy
	return r.db.Create(workInfo).Error
}

func (r *workInformationRepository) GetByID(id uint) (*domain.EmployeeWorkInformation, error) {
	var workInfo domain.EmployeeWorkInformation
	err := r.db.Preload("Employee").Preload("Company").Preload("Department").Preload("JobPosition").
		Where("id = ? AND deleted = ?", id, false).First(&workInfo).Error
	if err != nil {
		return nil, err
	}
	return &workInfo, nil
}

func (r *workInformationRepository) GetByEmployeeID(employeeID uint) ([]domain.EmployeeWorkInformation, error) {
	var workInfos []domain.EmployeeWorkInformation
	err := r.db.Preload("Employee").Preload("Company").Preload("Department").Preload("JobPosition").
		Where("employee_id = ? AND deleted = ?", employeeID, false).
		Order("start_date DESC").
		Find(&workInfos).Error
	return workInfos, err
}

func (r *workInformationRepository) GetByUserID(userID uint) ([]domain.EmployeeWorkInformation, error) {
	var workInfos []domain.EmployeeWorkInformation
	err := r.db.Preload("Employee").Preload("Company").Preload("Department").Preload("JobPosition").
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.employee_id", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees"), domain.GetTableName("hr_employee_work_information"))).
		Where(fmt.Sprintf("%s.user_id = ? AND %s.deleted = ? AND %s.deleted = ?", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employee_work_information"), domain.GetTableName("hr_employees")), userID, false, false).
		Order(fmt.Sprintf("%s.start_date DESC", domain.GetTableName("hr_employee_work_information"))).
		Find(&workInfos).Error
	return workInfos, err
}

func (r *workInformationRepository) GetAll(limit, offset int, sortParams types.SortParams, employeeID *uint) ([]domain.EmployeeWorkInformation, int64, error) {
	var workInformations []domain.EmployeeWorkInformation
	var total int64

	// Validate and sanitize sort field
	validSortFields := map[string]bool{
		"id":              true,
		"employee_id":     true,
		"company_id":      true,
		"department_id":   true,
		"job_position_id": true,
		"start_date":      true,
		"end_date":        true,
		"created_at":      true,
		"updated_at":      true,
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

	// Build base query
	query := r.db.Model(&domain.EmployeeWorkInformation{}).Where("deleted = ?", false)

	// Apply employee_id filter if provided
	if employeeID != nil {
		query = query.Where("employee_id = ?", *employeeID)
	}

	// Count total records
	query.Count(&total)

	// Get paginated records with sorting
	finalQuery := r.db.Preload("Employee").Preload("Company").Preload("Department").Preload("JobPosition").
		Where("deleted = ?", false)

	// Apply employee_id filter if provided
	if employeeID != nil {
		finalQuery = finalQuery.Where("employee_id = ?", *employeeID)
	}

	err := finalQuery.Order(orderBy).
		Limit(limit).Offset(offset).Find(&workInformations).Error

	return workInformations, total, err
}

func (r *workInformationRepository) Update(workInfo *domain.EmployeeWorkInformation, modifiedBy string) error {
	workInfo.ModifiedBy = modifiedBy
	return r.db.Save(workInfo).Error
}

func (r *workInformationRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.EmployeeWorkInformation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}
