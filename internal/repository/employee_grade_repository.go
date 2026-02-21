package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type EmployeeGradeRepository interface {
	Create(employeeGrade *domain.EmployeeGrade, createdBy string) error
	GetByID(id uint) (*domain.EmployeeGrade, error)
	GetByUserID(userID uint) ([]domain.EmployeeGrade, error)
	GetAll(limit, offset int, sortParams types.SortParams, employeeID *uint) ([]domain.EmployeeGrade, int64, error)
	Update(employeeGrade *domain.EmployeeGrade, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type employeeGradeRepository struct {
	db *gorm.DB
}

func NewEmployeeGradeRepository(db *gorm.DB) EmployeeGradeRepository {
	return &employeeGradeRepository{
		db: db,
	}
}

func (r *employeeGradeRepository) Create(employeeGrade *domain.EmployeeGrade, createdBy string) error {
	employeeGrade.CreatedBy = createdBy
	employeeGrade.ModifiedBy = createdBy
	return r.db.Create(employeeGrade).Error
}

func (r *employeeGradeRepository) GetByID(id uint) (*domain.EmployeeGrade, error) {
	var employeeGrade domain.EmployeeGrade
	err := r.db.Preload("Employee").Preload("Grade").
		Where("id = ? AND deleted = ?", id, false).First(&employeeGrade).Error
	if err != nil {
		return nil, err
	}
	return &employeeGrade, nil
}

func (r *employeeGradeRepository) GetByUserID(userID uint) ([]domain.EmployeeGrade, error) {
	var employeeGrades []domain.EmployeeGrade
	err := r.db.Preload("Employee").Preload("Grade").
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.employee_id", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees"), domain.GetTableName("hr_employee_grades"))).
		Where(fmt.Sprintf("%s.user_id = ? AND %s.deleted = ? AND %s.deleted = ?", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employee_grades"), domain.GetTableName("hr_employees")), userID, false, false).
		Order(fmt.Sprintf("%s.start_date DESC", domain.GetTableName("hr_employee_grades"))).
		Find(&employeeGrades).Error
	return employeeGrades, err
}

func (r *employeeGradeRepository) GetAll(limit, offset int, sortParams types.SortParams, employeeID *uint) ([]domain.EmployeeGrade, int64, error) {
	var employeeGrades []domain.EmployeeGrade
	var total int64

	// Validate and sanitize sort field
	validSortFields := map[string]bool{
		"id":          true,
		"employee_id": true,
		"grade_id":    true,
		"start_date":  true,
		"end_date":    true,
		"created_at":  true,
		"updated_at":  true,
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
	query := r.db.Model(&domain.EmployeeGrade{}).Where("deleted = ?", false)

	// Apply employee_id filter if provided
	if employeeID != nil {
		query = query.Where("employee_id = ?", *employeeID)
	}

	// Count total records
	query.Count(&total)

	// Get paginated records with sorting
	finalQuery := r.db.Preload("Employee").Preload("Grade").
		Where("deleted = ?", false)

	// Apply employee_id filter if provided
	if employeeID != nil {
		finalQuery = finalQuery.Where("employee_id = ?", *employeeID)
	}

	err := finalQuery.Order(orderBy).
		Limit(limit).Offset(offset).Find(&employeeGrades).Error

	return employeeGrades, total, err
}

func (r *employeeGradeRepository) Update(employeeGrade *domain.EmployeeGrade, modifiedBy string) error {
	employeeGrade.ModifiedBy = modifiedBy
	return r.db.Save(employeeGrade).Error
}

func (r *employeeGradeRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.EmployeeGrade{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}
