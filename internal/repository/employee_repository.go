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
	GetTotalCount() (int64, error)
	GetEmployeeCountByGender() ([]interface{}, error)
	GetEmployeeCountByPosition() ([]interface{}, error)
	GetEmployeeCountByCompanyDepartment() ([]interface{}, error)
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

// GetTotalCount returns the total number of employees
func (r *employeeRepository) GetTotalCount() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Employee{}).Where("deleted = ?", false).Count(&count).Error
	return count, err
}

// GetEmployeeCountByGender returns employee count grouped by gender
func (r *employeeRepository) GetEmployeeCountByGender() ([]interface{}, error) {
	type GenderCount struct {
		Gender string `json:"gender"`
		Count  int64  `json:"count"`
	}

	var results []GenderCount
	err := r.db.Model(&domain.Employee{}).
		Where("deleted = ?", false).
		Group("gender").
		Select("gender, COUNT(*) as count").
		Order("count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert to []interface{}
	var data []interface{}
	for _, result := range results {
		data = append(data, result)
	}
	return data, nil
}

// GetEmployeeCountByPosition returns employee count grouped by job position
func (r *employeeRepository) GetEmployeeCountByPosition() ([]interface{}, error) {
	type PositionCount struct {
		PositionTitle string `json:"position_title"`
		Count         int64  `json:"count"`
	}

	var results []PositionCount
	err := r.db.Model(&domain.Employee{}).
		Joins("JOIN hr_employee_work_information ON hr_employee_work_information.employee_id = hr_employees.id").
		Joins("JOIN hr_job_positions ON hr_job_positions.id = hr_employee_work_information.job_position_id").
		Where("hr_employees.deleted = ?", false).
		Group("hr_job_positions.title").
		Select("hr_job_positions.title as position_title, COUNT(*) as count").
		Order("count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert to []interface{}
	var data []interface{}
	for _, result := range results {
		data = append(data, result)
	}
	return data, nil
}

// GetEmployeeCountByCompanyDepartment returns employee count grouped by company and department
func (r *employeeRepository) GetEmployeeCountByCompanyDepartment() ([]interface{}, error) {
	type CompanyDepartmentCount struct {
		CompanyName    string `json:"company_name"`
		DepartmentName string `json:"department_name"`
		Count          int64  `json:"count"`
	}

	var results []CompanyDepartmentCount
	err := r.db.Model(&domain.Employee{}).
		Joins("JOIN hr_employee_work_information ON hr_employee_work_information.employee_id = hr_employees.id").
		Joins("JOIN hr_departments ON hr_departments.id = hr_employee_work_information.department_id").
		Joins("JOIN hr_companies ON hr_companies.id = hr_departments.company_id").
		Where("hr_employees.deleted = ?", false).
		Group("hr_companies.name, hr_departments.name").
		Select("hr_companies.name as company_name, hr_departments.name as department_name, COUNT(*) as count").
		Order("hr_companies.name ASC, hr_departments.name ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert to []interface{}
	var data []interface{}
	for _, result := range results {
		data = append(data, result)
	}
	return data, nil
}
