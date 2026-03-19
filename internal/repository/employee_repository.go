package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
	"strings"

	"gorm.io/gorm"
)

type EmployeeRepository interface {
	Create(employee *domain.Employee, createdBy string) error
	GetByID(id uint) (*domain.Employee, error)
	GetByUserID(userID uint) (*domain.Employee, error)
	GetByEmail(email string) (*domain.Employee, error)
	GetByIdentityNo(identityNo string) (*domain.Employee, error)
	GetByPhone(phone string) (*domain.Employee, error)
	GetByCompanyEmail(companyEmail string) (*domain.Employee, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Employee, int64, error)
	GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.Employee, int64, error)
	Update(employee *domain.Employee, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	GetTotalCount() (int64, error)
	GetTotalCountWithFilters(filters map[string]interface{}) (int64, error)
	GetEmployeeCountByGender() ([]interface{}, error)
	GetEmployeeCountByPosition() ([]interface{}, error)
	GetEmployeeCountByCompanyDepartment() ([]interface{}, error)
	GetEmployeeCountByGrade() ([]interface{}, error)
	GetWorkDayReportData(startDate, endDate string, companyID, departmentID *uint) ([]types.WorkDayReportRow, error)
	GetGradeReportData(companyID, departmentID *uint) ([]types.GradeReportRow, error)
}

type employeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) EmployeeRepository {
	return &employeeRepository{db: db}
}

func normalizeFilterValue(value interface{}) string {
	trimmed := strings.TrimSpace(fmt.Sprintf("%v", value))
	if trimmed == "" || strings.EqualFold(trimmed, "null") || strings.EqualFold(trimmed, "undefined") {
		return ""
	}
	return trimmed
}

func normalizedLikePattern(value interface{}) string {
	normalized := normalizeFilterValue(value)
	if normalized == "" {
		return ""
	}
	return "%" + normalized + "%"
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

func (r *employeeRepository) GetByCompanyEmail(companyEmail string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Preload("User").Where("company_email = ? AND deleted = ? AND status != ?", companyEmail, false, "PASSIVE").First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &employee, err
}

func (r *employeeRepository) GetByEmail(email string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Preload("User").Where("email = ? AND deleted = ? AND status != ?", email, false, "PASSIVE").First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &employee, err
}

func (r *employeeRepository) GetByIdentityNo(identityNo string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Preload("User").Where("identity_no = ? AND deleted = ? AND status != ?", identityNo, false, "PASSIVE").First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &employee, err
}

func (r *employeeRepository) GetByPhone(phone string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Preload("User").Where("phone = ? AND deleted = ? AND status != ?", phone, false, "PASSIVE").First(&employee).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
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

// GetAllWithFilters returns filtered employees with pagination and sorting
func (r *employeeRepository) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.Employee, int64, error) {
	var employees []*domain.Employee
	var total int64

	// Replace hardcoded table names with dynamic table names using domain.GetTableName
	query := r.db.Model(&domain.Employee{}).Where(fmt.Sprintf("%s.deleted = ?", domain.GetTableName("hr_employees")), false)

	// Apply filters
	if filters != nil {
		if firstName, ok := filters["first_name"]; ok {
			firstNameFilter := normalizedLikePattern(firstName)
			if firstNameFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.first_name) LIKE LOWER(?)", domain.GetTableName("hr_employees")), firstNameFilter)
			}
		}

		if email, ok := filters["email"]; ok {
			emailFilter := normalizedLikePattern(email)
			if emailFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.email) LIKE LOWER(?) OR LOWER(%s.company_email) LIKE LOWER(?)", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), emailFilter, emailFilter)
			}
		}

		if company, ok := filters["company"]; ok {
			query = query.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s d ON d.id = wi.department_id AND d.deleted = false
				JOIN %s c ON c.id = d.company_id AND c.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(c.name), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_departments"),
				domain.GetTableName("hr_companies"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", company)+"%")
		}

		if department, ok := filters["department"]; ok {
			query = query.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s d ON d.id = wi.department_id AND d.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(d.name), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_departments"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", department)+"%")
		}

		if jobTitle, ok := filters["job_title"]; ok {
			query = query.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s jp ON jp.id = wi.job_position_id AND jp.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(jp.title), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_job_positions"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", jobTitle)+"%")
		}

		if companyID, ok := filters["company_id"]; ok {
			query = query.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id AND %s.deleted = false`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employees"),
				domain.GetTableName("hr_employee_work_information"))).
				Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_departments"))).
				Where(fmt.Sprintf("%s.company_id = ?", domain.GetTableName("hr_departments")), companyID)
		}

		// Department IDs filter
		if departmentIDs, ok := filters["department_ids"]; ok {
			if departmentIDSlice, ok := departmentIDs.([]int); ok && len(departmentIDSlice) > 0 {
				// If company_id was not provided, we need to add the JOINs
				if _, hasCompany := filters["company_id"]; !hasCompany {
					query = query.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id AND %s.deleted = false`,
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employees"),
						domain.GetTableName("hr_employee_work_information"))).
						Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
							domain.GetTableName("hr_departments"),
							domain.GetTableName("hr_departments"),
							domain.GetTableName("hr_employee_work_information"),
							domain.GetTableName("hr_departments")))
				}
				query = query.Where(fmt.Sprintf("%s.department_id IN ?", domain.GetTableName("hr_employee_work_information")), departmentIDSlice)
			}
		}

		// Manager filter - needs work_information + department JOINs
		if manager, ok := filters["manager"]; ok {
			hasWorkInfoJoin := false
			hasDepartmentJoin := false
			if _, hasCompany := filters["company_id"]; hasCompany {
				hasWorkInfoJoin = true
				hasDepartmentJoin = true
			} else if _, hasDeptIDs := filters["department_ids"]; hasDeptIDs {
				hasWorkInfoJoin = true
			}
			if !hasWorkInfoJoin {
				query = query.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id AND %s.deleted = false`,
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employees"),
					domain.GetTableName("hr_employee_work_information")))
			}
			if !hasDepartmentJoin {
				query = query.Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_departments")))
			}
			managerFilter := normalizedLikePattern(manager)
			if managerFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.manager) LIKE LOWER(?)", domain.GetTableName("hr_departments")), managerFilter)
			}
		}

		if status, ok := filters["status"]; ok {
			query = query.Where(fmt.Sprintf("%s.status = ?", domain.GetTableName("hr_employees")), status)
		}

		if identityNo, ok := filters["identity_no"]; ok {
			identityNoFilter := normalizedLikePattern(identityNo)
			if identityNoFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.identity_no) LIKE LOWER(?)", domain.GetTableName("hr_employees")), identityNoFilter)
			}
		}

		if gender, ok := filters["gender"]; ok {
			query = query.Where(fmt.Sprintf("%s.gender = ?", domain.GetTableName("hr_employees")), gender)
		}

		if maritalStatus, ok := filters["marital_status"]; ok {
			query = query.Where(fmt.Sprintf("%s.marital_status = ?", domain.GetTableName("hr_employees")), maritalStatus)
		}

		if gradeID, ok := filters["grade_id"]; ok {
			query = query.Where(fmt.Sprintf("%s.grade_id = ?", domain.GetTableName("hr_employees")), gradeID)
		}

	}
	// Log the final SQL query using GORM's Statement.SQL.String()
	sql := query.Statement.SQL.String()
	fmt.Printf("Final SQL Query: %s\n", sql)

	// Count total records with same filters applied
	// Create a new query with same base conditions and filters
	countQuery := r.db.Model(&domain.Employee{}).
		Where(fmt.Sprintf("%s.deleted = ?", domain.GetTableName("hr_employees")), false)

	// Apply same filters to count query
	if filters != nil {
		if firstName, ok := filters["first_name"]; ok {
			firstNameFilter := normalizedLikePattern(firstName)
			if firstNameFilter != "" {
				countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.first_name) LIKE LOWER(?)", domain.GetTableName("hr_employees")), firstNameFilter)
			}
		}
		if email, ok := filters["email"]; ok {
			emailFilter := normalizedLikePattern(email)
			if emailFilter != "" {
				countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.email) LIKE LOWER(?) OR LOWER(%s.company_email) LIKE LOWER(?)", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), emailFilter, emailFilter)
			}
		}
		if company, ok := filters["company"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s d ON d.id = wi.department_id AND d.deleted = false
				JOIN %s c ON c.id = d.company_id AND c.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(c.name), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_departments"),
				domain.GetTableName("hr_companies"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", company)+"%")
		}
		if department, ok := filters["department"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s d ON d.id = wi.department_id AND d.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(d.name), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_departments"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", department)+"%")
		}
		if jobTitle, ok := filters["job_title"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s jp ON jp.id = wi.job_position_id AND jp.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(jp.title), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_job_positions"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", jobTitle)+"%")
		}
		if companyID, ok := filters["company_id"]; ok {
			countQuery = countQuery.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id AND %s.deleted = false`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employees"),
				domain.GetTableName("hr_employee_work_information"))).
				Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_departments"))).
				Where(fmt.Sprintf("%s.company_id = ?", domain.GetTableName("hr_departments")), companyID)
		}
		// Department IDs filter for count query
		if departmentIDs, ok := filters["department_ids"]; ok {
			if departmentIDSlice, ok := departmentIDs.([]int); ok && len(departmentIDSlice) > 0 {
				if _, hasCompany := filters["company_id"]; !hasCompany {
					countQuery = countQuery.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id AND %s.deleted = false`,
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employees"),
						domain.GetTableName("hr_employee_work_information"))).
						Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
							domain.GetTableName("hr_departments"),
							domain.GetTableName("hr_departments"),
							domain.GetTableName("hr_employee_work_information"),
							domain.GetTableName("hr_departments")))
				}
				countQuery = countQuery.Where(fmt.Sprintf("%s.department_id IN ?", domain.GetTableName("hr_employee_work_information")), departmentIDSlice)
			}
		}
		// Manager filter for count query
		if manager, ok := filters["manager"]; ok {
			if _, hasCompany := filters["company_id"]; !hasCompany {
				if _, hasDepartmentIDs := filters["department_ids"]; !hasDepartmentIDs {
					countQuery = countQuery.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id AND %s.deleted = false`,
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employees"),
						domain.GetTableName("hr_employee_work_information"))).
						Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
							domain.GetTableName("hr_departments"),
							domain.GetTableName("hr_departments"),
							domain.GetTableName("hr_employee_work_information"),
							domain.GetTableName("hr_departments")))
				}
			}
			managerFilter := normalizedLikePattern(manager)
			if managerFilter != "" {
				countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.manager) LIKE LOWER(?)", domain.GetTableName("hr_departments")), managerFilter)
			}
		}
		if status, ok := filters["status"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("%s.status = ?", domain.GetTableName("hr_employees")), status)
		}
		if identityNo, ok := filters["identity_no"]; ok {
			identityNoFilter := normalizedLikePattern(identityNo)
			if identityNoFilter != "" {
				countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.identity_no) LIKE LOWER(?)", domain.GetTableName("hr_employees")), identityNoFilter)
			}
		}
		if gender, ok := filters["gender"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("%s.gender = ?", domain.GetTableName("hr_employees")), gender)
		}
		if maritalStatus, ok := filters["marital_status"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("%s.marital_status = ?", domain.GetTableName("hr_employees")), maritalStatus)
		}
		if gradeID, ok := filters["grade_id"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("%s.grade_id = ?", domain.GetTableName("hr_employees")), gradeID)
		}
	}

	// Get the count
	countQuery.Count(&total)

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

	orderBy := fmt.Sprintf("%s.%s %s", domain.GetTableName("hr_employees"), sortField, direction)

	// Get paginated records with sorting and preloading
	err := query.Preload("User").
		Select(fmt.Sprintf("DISTINCT %s.*", domain.GetTableName("hr_employees"))).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&employees).Error

	return employees, total, err
}

// GetTotalCountWithFilters returns the total number of employees with filters applied
func (r *employeeRepository) GetTotalCountWithFilters(filters map[string]interface{}) (int64, error) {
	var count int64

	// Build the query with filters - explicitly specify table name for deleted column
	query := r.db.Model(&domain.Employee{}).Where(fmt.Sprintf("%s.deleted = ?", domain.GetTableName("hr_employees")), false)

	// Apply filters (same logic as GetAllWithFilters)
	if filters != nil {
		// First name filter (LIKE search)
		if firstName, ok := filters["first_name"]; ok {
			firstNameFilter := normalizedLikePattern(firstName)
			if firstNameFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.first_name) LIKE LOWER(?)", domain.GetTableName("hr_employees")), firstNameFilter)
			}
		}

		// Email filter (LIKE search for both personal and company email)
		if email, ok := filters["email"]; ok {
			emailFilter := normalizedLikePattern(email)
			if emailFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.email) LIKE LOWER(?) OR LOWER(%s.company_email) LIKE LOWER(?)", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), emailFilter, emailFilter)
			}
		}

		// Company name filter (case-insensitive LIKE)
		if company, ok := filters["company"]; ok {
			query = query.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s d ON d.id = wi.department_id AND d.deleted = false
				JOIN %s c ON c.id = d.company_id AND c.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(c.name), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_departments"),
				domain.GetTableName("hr_companies"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", company)+"%")
		}

		// Department name filter (case-insensitive LIKE)
		if department, ok := filters["department"]; ok {
			query = query.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s d ON d.id = wi.department_id AND d.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(d.name), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_departments"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", department)+"%")
		}

		// Job title filter (case-insensitive LIKE)
		if jobTitle, ok := filters["job_title"]; ok {
			query = query.Where(fmt.Sprintf(`EXISTS (
				SELECT 1 FROM %s wi
				JOIN %s jp ON jp.id = wi.job_position_id AND jp.deleted = false
				WHERE wi.employee_id = %s.id
				AND wi.deleted = false
				AND regexp_replace(LOWER(jp.title), '\\s+', '', 'g') LIKE regexp_replace(LOWER(?), '\\s+', '', 'g')
			)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_job_positions"),
				domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", jobTitle)+"%")
		}

		// Identity number filter
		if identityNo, ok := filters["identity_no"]; ok {
			identityNoFilter := normalizedLikePattern(identityNo)
			if identityNoFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.identity_no) LIKE LOWER(?)", domain.GetTableName("hr_employees")), identityNoFilter)
			}
		}

		// Gender filter
		if gender, ok := filters["gender"]; ok {
			query = query.Where(fmt.Sprintf("%s.gender = ?", domain.GetTableName("hr_employees")), gender)
		}

		// Marital status filter
		if maritalStatus, ok := filters["marital_status"]; ok {
			query = query.Where(fmt.Sprintf("%s.marital_status = ?", domain.GetTableName("hr_employees")), maritalStatus)
		}

		// Status filter (ACTIVE/PASSIVE)
		if status, ok := filters["status"]; ok {
			query = query.Where(fmt.Sprintf("%s.status = ?", domain.GetTableName("hr_employees")), status)
		}

		// Status filter (ACTIVE/PASSIVE)
		if status, ok := filters["status"]; ok {
			query = query.Where(fmt.Sprintf("%s.status = ?", domain.GetTableName("hr_employees")), status)
		}

		// Grade ID filter
		if gradeID, ok := filters["grade_id"]; ok {
			query = query.Where(fmt.Sprintf("%s.grade_id = ?", domain.GetTableName("hr_employees")), gradeID)
		}

		// Company and Department filters need JOIN with work information
		if companyID, ok := filters["company_id"]; ok {
			query = query.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id 
				AND %s.deleted = false 
				AND %s.id = (
					SELECT id FROM %s 
					WHERE employee_id = %s.id 
					AND deleted = false 
					ORDER BY start_date DESC 
					LIMIT 1
				)`,
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employees"),
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employee_work_information"),
				domain.GetTableName("hr_employees"))).
				Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_departments"))).
				Where(fmt.Sprintf("%s.company_id = ?", domain.GetTableName("hr_departments")), companyID)
		}

		// Handle multiple department IDs - only check current/latest department
		if departmentIDs, ok := filters["department_ids"]; ok {
			if departmentIDSlice, ok := departmentIDs.([]int); ok && len(departmentIDSlice) > 0 {
				// If we haven't joined yet (no company filter), do the join with latest work information
				if _, hasCompany := filters["company_id"]; !hasCompany {
					query = query.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id 
						AND %s.deleted = false 
						AND %s.id = (
							SELECT id FROM %s 
							WHERE employee_id = %s.id 
							AND deleted = false 
							ORDER BY start_date DESC 
							LIMIT 1
						)`,
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employees"),
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employee_work_information"),
						domain.GetTableName("hr_employees")))
				}
				query = query.Where(fmt.Sprintf("%s.department_id IN ?", domain.GetTableName("hr_employee_work_information")), departmentIDSlice)
			}
		} else if departmentID, ok := filters["department_id"]; ok {
			// Handle single department ID for backward compatibility - only check current/latest department
			// If we haven't joined yet (no company filter), do the join with latest work information
			if _, hasCompany := filters["company_id"]; !hasCompany {
				query = query.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id 
					AND %s.deleted = false 
					AND %s.id = (
						SELECT id FROM %s 
						WHERE employee_id = %s.id 
						AND deleted = false 
						ORDER BY start_date DESC 
						LIMIT 1
					)`,
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employees"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employees")))
			}
			query = query.Where(fmt.Sprintf("%s.department_id = ?", domain.GetTableName("hr_employee_work_information")), departmentID)
		}

		// Manager filter needs JOIN with departments - only check current/latest department
		if manager, ok := filters["manager"]; ok {
			// Always ensure we have the necessary JOINs for manager filter
			hasWorkInfoJoin := false
			hasDepartmentJoin := false

			// Check if we already have work info join from previous filters
			if _, hasCompany := filters["company_id"]; hasCompany {
				hasWorkInfoJoin = true
				hasDepartmentJoin = true
			} else if _, hasDept := filters["department_id"]; hasDept {
				hasWorkInfoJoin = true
			} else if _, hasDeptIDs := filters["department_ids"]; hasDeptIDs {
				hasWorkInfoJoin = true
			}

			// Add work info join if not already present
			if !hasWorkInfoJoin {
				query = query.Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id 
						AND %s.deleted = false 
						AND %s.id = (
							SELECT id FROM %s 
							WHERE employee_id = %s.id 
							AND deleted = false 
							ORDER BY start_date DESC 
							LIMIT 1
						)`,
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employees"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_employees")))
			}

			// Add department join if not already present
			if !hasDepartmentJoin {
				query = query.Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_departments"),
					domain.GetTableName("hr_employee_work_information"),
					domain.GetTableName("hr_departments")))
			}

			// Apply manager filter
			managerFilter := normalizedLikePattern(manager)
			if managerFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.manager) LIKE LOWER(?)", domain.GetTableName("hr_departments")), managerFilter)
			}
		}
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *employeeRepository) Update(employee *domain.Employee, modifiedBy string) error {
	employee.ModifiedBy = modifiedBy

	// Use Updates with a map to explicitly update all fields, including nil dates
	updates := map[string]interface{}{
		"email":                      employee.Email,
		"company_email":              employee.CompanyEmail,
		"first_name":                 employee.FirstName,
		"last_name":                  employee.LastName,
		"phone":                      employee.Phone,
		"address":                    employee.Address,
		"state":                      employee.State,
		"city":                       employee.City,
		"gender":                     employee.Gender,
		"date_of_birth":              employee.DateOfBirth,
		"hire_date":                  employee.HireDate,
		"leave_date":                 employee.LeaveDate,
		"total_gap":                  employee.TotalGap,
		"marital_status":             employee.MaritalStatus,
		"emergency_contact":          employee.EmergencyContact,
		"emergency_contact_name":     employee.EmergencyContactName,
		"emergency_contact_relation": employee.EmergencyContactRelation,
		"grade_id":                   employee.GradeID,
		"contract_no":                employee.ContractNo,
		"profession_start_date":      employee.ProfessionStartDate,
		"note":                       employee.Note,
		"mother_name":                employee.MotherName,
		"father_name":                employee.FatherName,
		"nationality":                employee.Nationality,
		"identity_no":                employee.IdentityNo,
		"status":                     employee.Status,
		"modified_by":                modifiedBy,
	}

	return r.db.Where("deleted = ?", false).Model(employee).Updates(updates).Error
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
		Where("deleted = ? AND status = ?", false, "ACTIVE").
		Where("deleted = ? AND status = ?", false, "ACTIVE").
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
		Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id 
			AND %s.deleted = false 
			AND %s.id = (
				SELECT id FROM %s 
				WHERE employee_id = %s.id 
				AND deleted = false 
				ORDER BY start_date DESC 
				LIMIT 1
			)`,
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employees"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employees"))).
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.job_position_id AND %s.deleted = false",
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_job_positions"))).
		Where(fmt.Sprintf("%s.deleted = ? AND %s.status = ?", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), false, "ACTIVE").
		Group(fmt.Sprintf("%s.title", domain.GetTableName("hr_job_positions"))).
		Select(fmt.Sprintf("%s.title as position_title, COUNT(*) as count", domain.GetTableName("hr_job_positions"))).
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
		Joins(fmt.Sprintf(`JOIN %s ON %s.employee_id = %s.id 
			AND %s.deleted = false 
			AND %s.id = (
				SELECT id FROM %s 
				WHERE employee_id = %s.id 
				AND deleted = false 
				ORDER BY start_date DESC 
				LIMIT 1
			)`,
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employees"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employees"))).
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id AND %s.deleted = false",
			domain.GetTableName("hr_departments"),
			domain.GetTableName("hr_departments"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_departments"))).
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.company_id AND %s.deleted = false",
			domain.GetTableName("hr_companies"),
			domain.GetTableName("hr_companies"),
			domain.GetTableName("hr_departments"),
			domain.GetTableName("hr_companies"))).
		Where(fmt.Sprintf("%s.deleted = ? AND %s.status = ?", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), false, "ACTIVE").
		Group(fmt.Sprintf("%s.name, %s.name", domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"))).
		Select(fmt.Sprintf("%s.name as company_name, %s.name as department_name, COUNT(*) as count",
			domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"))).
		Order(fmt.Sprintf("%s.name ASC, %s.name ASC", domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"))).
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

// GetEmployeeCountByGrade returns employee count grouped by grade
func (r *employeeRepository) GetEmployeeCountByGrade() ([]interface{}, error) {
	type GradeCount struct {
		GradeName string `json:"grade_name"`
		Count     int64  `json:"count"`
	}

	var results []GradeCount
	err := r.db.Model(&domain.Employee{}).
		Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.grade_id AND %s.deleted = false",
			domain.GetTableName("hr_grades"),
			domain.GetTableName("hr_grades"),
			domain.GetTableName("hr_employees"),
			domain.GetTableName("hr_grades"))).
		Where(fmt.Sprintf("%s.deleted = ? AND %s.status = ?", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), false, "ACTIVE").
		Group("COALESCE(" + domain.GetTableName("hr_grades") + ".name, 'Bilinmiyor')").
		Select("COALESCE(" + domain.GetTableName("hr_grades") + ".name, 'Bilinmiyor') as grade_name, COUNT(*) as count").
		Order("count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	var data []interface{}
	for _, result := range results {
		data = append(data, result)
	}
	return data, nil
}

// GetWorkDayReportData executes the work day report SQL query
func (r *employeeRepository) GetWorkDayReportData(startDate, endDate string, companyID, departmentID *uint) ([]types.WorkDayReportRow, error) {
	var rows []types.WorkDayReportRow

	// Build base query with numbered parameters for PostgreSQL
	query := `
		WITH date_range AS (
			SELECT generate_series(
				$1::date,
				$2::date,
				INTERVAL '1 day'
			)::date AS work_date
		),
		
		working_days AS (
			SELECT work_date
			FROM date_range
			WHERE EXTRACT(DOW FROM work_date) NOT IN (0,6)
			  AND NOT EXISTS (
					SELECT 1
					FROM hr_holidays h
					WHERE h.holiday_date = date_range.work_date
			  )
		),
		
		employee_days AS (
			SELECT
				e.id AS employee_id,
				w.work_date,
				wi.start_date AS team_start_date,
				wi.end_date   AS team_end_date,
				wi.company_id,
				wi.department_id,
				CASE
					WHEN lr.id IS NOT NULL THEN 'LEAVE'
					ELSE 'WORK'
					END AS day_type
			FROM working_days w
				CROSS JOIN hr_employees e
			
				LEFT JOIN hr_employee_work_information wi
					ON wi.employee_id = e.id AND wi.deleted = false
						AND w.work_date BETWEEN wi.start_date
						   AND COALESCE(wi.end_date, DATE '9999-12-31')
			
				LEFT JOIN hr_leave_requests lr
					ON lr.employee_id = e.id AND lr.deleted = false
						AND w.work_date BETWEEN lr.start_date AND lr.end_date
						AND lr.status = 'APPROVED'
			
			WHERE
				w.work_date BETWEEN e.hire_date
				AND COALESCE(e.leave_date, DATE '9999-12-31')
			  AND wi.id IS NOT NULL
		)
		
		SELECT
			e.id,
			e.identity_no,
			e.first_name,
			e.last_name,
			
			c.id   AS company_id,
			c.name AS company_name,
			
			d.id   AS department_id,
			d.name AS department_name,
			d.manager AS manager,

			MIN(ed.team_start_date) AS team_start_date,
			MAX(ed.team_end_date)   AS team_end_date,
			
			e.hire_date,
			e.leave_date,
		
			COUNT(CASE WHEN ed.day_type = 'WORK' THEN 1 END) AS worked_days,
			COUNT(CASE WHEN ed.day_type = 'LEAVE' THEN 1 END) AS used_leave_days,

			COUNT(CASE WHEN ed.day_type = 'WORK' THEN 1 END)
				+
			COUNT(CASE WHEN ed.day_type = 'LEAVE' THEN 1 END) AS work_days
		
		FROM employee_days ed
		
		JOIN hr_employees e ON e.id = ed.employee_id
		
		LEFT JOIN hr_companies c ON c.id = ed.company_id
		LEFT JOIN hr_departments d ON d.id = ed.department_id
		
		WHERE 1=1`

	// Build dynamic parameters list
	params := []interface{}{startDate, endDate}
	paramCounter := 3

	// Add company filter if provided
	if companyID != nil {
		query += fmt.Sprintf("\n\t\t\tAND c.id = $%d", paramCounter)
		params = append(params, *companyID)
		paramCounter++
	}

	// Add department filter if provided
	if departmentID != nil {
		query += fmt.Sprintf("\n\t\t\tAND d.id = $%d", paramCounter)
		params = append(params, *departmentID)
		paramCounter++
	}

	// Add GROUP BY and ORDER BY
	query += `
		GROUP BY
			e.id,
			e.identity_no,
			e.first_name,
			e.last_name,
			c.id,
			c.name,
			d.id,
			d.name,
			e.hire_date,
			e.leave_date
		ORDER BY e.id
	`

	err := r.db.Raw(query, params...).Scan(&rows).Error
	return rows, err
}

// GetGradeReportData executes the grade report SQL query
func (r *employeeRepository) GetGradeReportData(companyID, departmentID *uint) ([]types.GradeReportRow, error) {
	var rows []types.GradeReportRow

	// Build base query with user-provided SQL
	query := `
WITH experience_calc AS (
    SELECT
        e.id,
        e.first_name,
        e.last_name,
        e.hire_date,
        e.profession_start_date,
        COALESCE(e.total_gap,0) AS total_gap,

        CASE
            WHEN e.profession_start_date IS NULL THEN 0
            ELSE
                (
                    EXTRACT(YEAR FROM AGE(CURRENT_DATE, e.profession_start_date))
                        +
                    EXTRACT(MONTH FROM AGE(CURRENT_DATE, e.profession_start_date)) / 12.0
                    )
                    - COALESCE(e.total_gap,0)
            END AS total_experience,

        -- rapor için text format
        CASE
            WHEN e.profession_start_date IS NULL THEN '0 Yıl 0 Ay'
            ELSE
                (
                    (
                        EXTRACT(YEAR FROM AGE(CURRENT_DATE, e.profession_start_date))
                        )::int
                        || ' Yıl '
                        ||
                    EXTRACT(MONTH FROM AGE(CURRENT_DATE, e.profession_start_date))::int
                        || ' Ay'
                    )
            END AS total_experience_text


    FROM hr_employees e
    WHERE e.deleted = false
      AND e.leave_date IS NULL
),

expected_grade_calc AS (
    SELECT
        ec.*,
        (ec.total_experience + 0.5) AS expected_experience
    FROM experience_calc ec
),

expected_grade AS (
    SELECT
        egc.*,
        g.id AS expected_grade_id,
        g.name AS expected_grade
    FROM expected_grade_calc egc
    LEFT JOIN hr_test_grades g
        ON egc.expected_experience >= g.min_year
            AND egc.expected_experience < g.max_year
),

current_grade AS (
    SELECT
        eg.employee_id,
        g.name AS current_grade
    FROM hr_employee_grades eg
    LEFT JOIN hr_grades g
        ON g.id = eg.grade_id
    WHERE eg.deleted = false
      AND eg.start_date <= CURRENT_DATE
      AND (eg.end_date IS NULL OR eg.end_date >= CURRENT_DATE)
),

team_info AS (
    SELECT DISTINCT ON (wi.employee_id)
        wi.employee_id,
        wi.start_date,
        wi.company_id,
        wi.department_id,
        c.name AS company_name,
        d.name AS department_name,
        d.manager
    FROM hr_employee_work_information wi
    LEFT JOIN hr_companies c
        ON c.id = wi.company_id
    LEFT JOIN hr_departments d
        ON d.id = wi.department_id
    WHERE wi.start_date <= CURRENT_DATE
      AND (wi.end_date IS NULL OR wi.end_date >= CURRENT_DATE)
    ORDER BY wi.employee_id, wi.start_date DESC
)
SELECT
    e.id,
    e.first_name,
    e.last_name,
    e.hire_date,
    t.company_name,
    t.department_name,
    t.manager,
    t.start_date AS team_start_date,
    e.profession_start_date,
    e.total_gap,
    e.total_experience_text,
    cg.current_grade,
    eg.expected_grade

FROM expected_grade eg
JOIN experience_calc e ON e.id = eg.id
LEFT JOIN current_grade cg ON cg.employee_id = e.id
LEFT JOIN team_info t ON t.employee_id = e.id
WHERE 1=1`

	// Build dynamic parameters list
	params := []interface{}{}
	paramCounter := 1

	// Add company filter if provided
	if companyID != nil {
		query += fmt.Sprintf("\n  AND t.company_id = $%d", paramCounter)
		params = append(params, *companyID)
		paramCounter++
	}

	// Add department filter if provided
	if departmentID != nil {
		query += fmt.Sprintf("\n  AND t.department_id = $%d", paramCounter)
		params = append(params, *departmentID)
		paramCounter++
	}

	// Complete the query
	query += `
ORDER BY e.id`

	err := r.db.Raw(query, params...).Scan(&rows).Error
	return rows, err
}
