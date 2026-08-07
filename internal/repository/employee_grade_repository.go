package repository

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EmployeeGradeRepository interface {
	Create(employeeGrade *domain.EmployeeGrade, createdBy string) error
	GetByID(id uint) (*domain.EmployeeGrade, error)
	GetByUserID(userID uint) ([]domain.EmployeeGrade, error)
	GetAll(limit, offset int, sortParams types.SortParams, employeeID *uint) ([]domain.EmployeeGrade, int64, error)
	Update(employeeGrade *domain.EmployeeGrade, modifiedBy string) error
	Delete(id uint, deletedBy string) error

	// Transaction runs fn with a repository bound to the same DB transaction.
	Transaction(fn func(txRepo EmployeeGradeRepository) error) error
	// GetActiveByEmployeeIDForUpdate locks the ACTIVE row (FOR UPDATE) for an employee.
	// Returns (nil, nil) when no active grade exists.
	GetActiveByEmployeeIDForUpdate(employeeID uint) (*domain.EmployeeGrade, error)
	// CloseActiveAsInactive sets status=INACTIVE and end_date on an ACTIVE row.
	CloseActiveAsInactive(id uint, endDate time.Time, modifiedBy string) error
	// ExistsByEmployeeGradeStartDate reports a non-deleted row with the same triple.
	ExistsByEmployeeGradeStartDate(employeeID, gradeID uint, startDate time.Time) (bool, error)
}

type employeeGradeRepository struct {
	db *gorm.DB
}

func NewEmployeeGradeRepository(db *gorm.DB) EmployeeGradeRepository {
	return &employeeGradeRepository{
		db: db,
	}
}

func (r *employeeGradeRepository) Transaction(fn func(txRepo EmployeeGradeRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&employeeGradeRepository{db: tx})
	})
}

func (r *employeeGradeRepository) Create(employeeGrade *domain.EmployeeGrade, createdBy string) error {
	employeeGrade.CreatedBy = createdBy
	employeeGrade.ModifiedBy = createdBy
	if employeeGrade.Status == "" {
		employeeGrade.Status = domain.EmployeeGradeStatusFromEndDate(employeeGrade.EndDate)
	}
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

	query := r.db.Model(&domain.EmployeeGrade{}).Where("deleted = ?", false)

	if employeeID != nil {
		query = query.Where("employee_id = ?", *employeeID)
	}

	query.Count(&total)

	finalQuery := r.db.Preload("Employee").Preload("Grade").
		Where("deleted = ?", false)

	if employeeID != nil {
		finalQuery = finalQuery.Where("employee_id = ?", *employeeID)
	}

	err := finalQuery.Order(orderBy).
		Limit(limit).Offset(offset).Find(&employeeGrades).Error

	return employeeGrades, total, err
}

func (r *employeeGradeRepository) Update(employeeGrade *domain.EmployeeGrade, modifiedBy string) error {
	employeeGrade.ModifiedBy = modifiedBy
	if employeeGrade.Status == "" {
		employeeGrade.Status = domain.EmployeeGradeStatusFromEndDate(employeeGrade.EndDate)
	}
	return r.db.Model(&domain.EmployeeGrade{}).
		Where("id = ? AND deleted = ?", employeeGrade.ID, false).
		Updates(map[string]interface{}{
			"grade_id":    employeeGrade.GradeID,
			"start_date":  employeeGrade.StartDate,
			"end_date":    employeeGrade.EndDate,
			"status":      employeeGrade.Status,
			"modified_by": modifiedBy,
		}).Error
}

func (r *employeeGradeRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.EmployeeGrade{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted":     true,
			"status":      domain.EmployeeGradeStatusInactive,
			"end_date":    gorm.Expr("COALESCE(end_date, start_date)"),
			"modified_by": deletedBy,
		}).Error
}

func (r *employeeGradeRepository) GetActiveByEmployeeIDForUpdate(employeeID uint) (*domain.EmployeeGrade, error) {
	var employeeGrade domain.EmployeeGrade
	err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("employee_id = ? AND deleted = ? AND status = ?", employeeID, false, domain.EmployeeGradeStatusActive).
		First(&employeeGrade).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &employeeGrade, nil
}

func (r *employeeGradeRepository) CloseActiveAsInactive(id uint, endDate time.Time, modifiedBy string) error {
	result := r.db.Model(&domain.EmployeeGrade{}).
		Where("id = ? AND deleted = ? AND status = ?", id, false, domain.EmployeeGradeStatusActive).
		Updates(map[string]interface{}{
			"status":      domain.EmployeeGradeStatusInactive,
			"end_date":    endDate,
			"modified_by": modifiedBy,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("failed to close active employee grade %d: no matching ACTIVE row", id)
	}
	return nil
}

func (r *employeeGradeRepository) ExistsByEmployeeGradeStartDate(employeeID, gradeID uint, startDate time.Time) (bool, error) {
	startDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	var count int64
	err := r.db.Model(&domain.EmployeeGrade{}).
		Where("employee_id = ? AND grade_id = ? AND deleted = ? AND start_date = ?", employeeID, gradeID, false, startDay).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
