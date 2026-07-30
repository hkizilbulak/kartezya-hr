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
	GetByIDs(ids []uint) ([]*domain.Employee, error)
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
	GetInternCountByCompanyDepartment() ([]interface{}, error)
	GetEmployeeCountByGrade() ([]interface{}, error)
	GetWorkDayReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.WorkDayReportRow, error)
	GetGradeReportData(companyID *uint, departmentIDs []uint, isActive *bool) ([]types.GradeReportRow, error)
	GetContractReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.ContractReportRow, error)
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

func (r *employeeRepository) GetByIDs(ids []uint) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	err := r.db.Where("id IN ?", ids).Find(&employees).Error
	return employees, err
}

func (r *employeeRepository) GetByID(id uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := preloadActiveEmployeeGrade(r.db.Preload("User")).
		Where("deleted = ?", false).First(&employee, id).Error
	return &employee, err
}

func (r *employeeRepository) GetByUserID(userID uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := preloadActiveEmployeeGrade(r.db.Preload("User")).
		Where("user_id = ? AND deleted = ?", userID, false).First(&employee).Error
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

	// Validate and sanitize sort field using shared employee list allowlist
	orderBy := buildEmployeeListOrderClause(sortParams.Sort, sortParams.Direction)

	// Count total records
	r.db.Model(&domain.Employee{}).Where("deleted = ?", false).Count(&total)

	// Get paginated records with sorting — all relations preloaded in bulk (eliminates N+1)
	err := preloadActiveEmployeeGrade(r.db.Preload("User").
		Preload("User.UserRoles").
		Preload("User.UserRoles.Role").
		Preload("EmployeeWorkInformation", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted = ?", false).Order("start_date DESC").Limit(1)
		}).
		Preload("EmployeeWorkInformation.Company").
		Preload("EmployeeWorkInformation.Department").
		Preload("EmployeeWorkInformation.JobPosition")).
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
			query = applyActiveEmployeeGradeIDFilter(query, domain.GetTableName("hr_employees"), gradeID)
		}

		// City filter (il)
		if city, ok := filters["city"]; ok {
			cityFilter := normalizedLikePattern(city)
			if cityFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.city) LIKE LOWER(?)", domain.GetTableName("hr_employees")), cityFilter)
			}
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
			countQuery = applyActiveEmployeeGradeIDFilter(countQuery, domain.GetTableName("hr_employees"), gradeID)
		}

		// City filter (il) for count query
		if city, ok := filters["city"]; ok {
			cityFilter := normalizedLikePattern(city)
			if cityFilter != "" {
				countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.city) LIKE LOWER(?)", domain.GetTableName("hr_employees")), cityFilter)
			}
		}
	}

	// Get the count
	countQuery.Count(&total)

	orderBy := buildEmployeeListOrderClause(sortParams.Sort, sortParams.Direction)

	// GROUP BY primary key collapses duplicate rows from filter JOINs while still
	// allowing ORDER BY correlated display-field subqueries (unlike SELECT DISTINCT).
	err := preloadActiveEmployeeGrade(query.Preload("User").
		Preload("User.UserRoles").
		Preload("User.UserRoles.Role").
		Preload("EmployeeWorkInformation", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted = ?", false).Order("start_date DESC").Limit(1)
		}).
		Preload("EmployeeWorkInformation.Company").
		Preload("EmployeeWorkInformation.Department").
		Preload("EmployeeWorkInformation.JobPosition")).
		Select(fmt.Sprintf("%s.*", domain.GetTableName("hr_employees"))).
		Group(fmt.Sprintf("%s.id", domain.GetTableName("hr_employees"))).
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
			query = applyActiveEmployeeGradeIDFilter(query, domain.GetTableName("hr_employees"), gradeID)
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

		// City filter (il) — must match GetAllWithFilters list/count city logic
		if city, ok := filters["city"]; ok {
			cityFilter := normalizedLikePattern(city)
			if cityFilter != "" {
				query = query.Where(fmt.Sprintf("LOWER(%s.city) LIKE LOWER(?)", domain.GetTableName("hr_employees")), cityFilter)
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
		// grade_id intentionally omitted: current grade lives on hr_employee_grades (ACTIVE).
		"contract_no":           employee.ContractNo,
		"profession_start_date": employee.ProfessionStartDate,
		"note":                  employee.Note,
		"mother_name":           employee.MotherName,
		"father_name":           employee.FatherName,
		"nationality":           employee.Nationality,
		"identity_no":           employee.IdentityNo,
		"status":                employee.Status,
		"modified_by":           modifiedBy,
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
		EmployeeNames  string `json:"employee_names"`
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
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.job_position_id AND %s.deleted = false",
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_job_positions"))).
		Where(fmt.Sprintf("%s.deleted = ? AND %s.status = ? AND LOWER(%s.title) NOT LIKE ? AND LOWER(%s.title) NOT LIKE ?",
			domain.GetTableName("hr_employees"),
			domain.GetTableName("hr_employees"),
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_job_positions")), false, "ACTIVE", "%intern%", "%stajyer%").
		Group(fmt.Sprintf("%s.name, %s.name", domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"))).
		Select(fmt.Sprintf("%s.name as company_name, %s.name as department_name, COUNT(*) as count, STRING_AGG(CONCAT(%s.first_name, ' ', %s.last_name), ', ') as employee_names",
			domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"), domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees"))).
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

// GetInternCountByCompanyDepartment returns active interns count grouped by company and department
func (r *employeeRepository) GetInternCountByCompanyDepartment() ([]interface{}, error) {
	type CompanyDepartmentCount struct {
		CompanyName    string `json:"company_name"`
		DepartmentName string `json:"department_name"`
		Count          int64  `json:"count"`
		EmployeeNames  string `json:"employee_names"`
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
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.job_position_id AND %s.deleted = false",
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_job_positions"))).
		Where(fmt.Sprintf("%s.deleted = ? AND %s.status = ? AND (LOWER(%s.title) LIKE ? OR LOWER(%s.title) LIKE ?)",
			domain.GetTableName("hr_employees"),
			domain.GetTableName("hr_employees"),
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_job_positions")),
			false, "ACTIVE", "%intern%", "%stajyer%").
		Group(fmt.Sprintf("%s.name, %s.name", domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"))).
		Select(fmt.Sprintf("%s.name as company_name, %s.name as department_name, COUNT(*) as count, STRING_AGG(CONCAT(%s.first_name, ' ', %s.last_name), ', ') as employee_names",
			domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"), domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees"))).
		Order(fmt.Sprintf("%s.name ASC, %s.name ASC", domain.GetTableName("hr_companies"), domain.GetTableName("hr_departments"))).
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

// GetEmployeeCountByGrade returns employee count grouped by ACTIVE EmployeeGrade.
// employees.grade_id is not used. Employees without an ACTIVE grade are counted as "Bilinmiyor".
func (r *employeeRepository) GetEmployeeCountByGrade() ([]interface{}, error) {
	type GradeCount struct {
		GradeName string `json:"grade_name"`
		Count     int64  `json:"count"`
	}

	employees := domain.GetTableName("hr_employees")
	grades := domain.GetTableName("hr_grades")
	employeeGrades := domain.GetTableName("hr_employee_grades")

	var results []GradeCount
	err := r.db.Model(&domain.Employee{}).
		Joins(fmt.Sprintf(
			"LEFT JOIN %s eg ON %s",
			employeeGrades,
			ActiveEmployeeGradeJoinOn("eg", employees+".id"),
		)).
		Joins(fmt.Sprintf(
			"LEFT JOIN %s g ON g.id = eg.grade_id AND g.deleted = false",
			grades,
		)).
		Where(fmt.Sprintf("%s.deleted = ? AND %s.status = ?", employees, employees), false, "ACTIVE").
		Group("COALESCE(g.name, 'Bilinmiyor')").
		Select("COALESCE(g.name, 'Bilinmiyor') as grade_name, COUNT(DISTINCT " + employees + ".id) as count").
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

// GetWorkDayReportData executes the work day report SQL query.
// current_grade is ACTIVE EmployeeGrade SoT (not date-window history).
// Work/leave day counting remains historical over [startDate, endDate].
func (r *employeeRepository) GetWorkDayReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.WorkDayReportRow, error) {
	var rows []types.WorkDayReportRow

	employees := domain.GetTableName("hr_employees")
	holidays := domain.GetTableName("hr_holidays")
	workInfo := domain.GetTableName("hr_employee_work_information")
	leaveRequests := domain.GetTableName("hr_leave_requests")
	companies := domain.GetTableName("hr_companies")
	departments := domain.GetTableName("hr_departments")
	currentGradeSelect := buildActiveCurrentGradeSelectSQL()

	query := fmt.Sprintf(`
		WITH date_range AS (
			SELECT generate_series(
				$1::date,
				$2::date,
				INTERVAL '1 day'
			)::date AS work_date
		),
		
		working_days AS (
			SELECT date_range.work_date,
                                CASE 
                                        WHEN h.id IS NOT NULL AND h.is_full_day = false THEN 0.5
                                        ELSE 1.0
                                END AS day_value
                        FROM date_range
                        LEFT JOIN %s h ON h.holiday_date = date_range.work_date
                        WHERE EXTRACT(DOW FROM date_range.work_date) NOT IN (0,6)
                          AND (h.id IS NULL OR h.is_full_day = false)
		),
		
		employee_days AS (
			SELECT
				e.id AS employee_id,
                                w.work_date,
                                w.day_value,
                                wi.start_date AS team_start_date,
				wi.end_date   AS team_end_date,
				wi.company_id,
				wi.department_id,
				CASE
					WHEN lr.id IS NOT NULL THEN 'LEAVE'
					ELSE 'WORK'
					END AS day_type
			FROM working_days w
				CROSS JOIN %s e
			
				LEFT JOIN %s wi
					ON wi.employee_id = e.id AND wi.deleted = false
						AND w.work_date BETWEEN wi.start_date
						   AND COALESCE(wi.end_date, DATE '9999-12-31')
			
				LEFT JOIN %s lr
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
			cg.current_grade,
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
		
			COALESCE(SUM(CASE WHEN ed.day_type = 'WORK' THEN ed.day_value ELSE 0 END), 0) AS worked_days,
                        COALESCE(SUM(CASE WHEN ed.day_type = 'LEAVE' THEN ed.day_value ELSE 0 END), 0) AS used_leave_days,
                        COALESCE(SUM(ed.day_value), 0) AS work_days
		
		FROM employee_days ed
		
		JOIN %s e ON e.id = ed.employee_id
		
		LEFT JOIN %s c ON c.id = ed.company_id
		LEFT JOIN %s d ON d.id = ed.department_id
		
		LEFT JOIN (
%s
		) cg ON cg.employee_id = e.id
		
		WHERE 1=1`,
		holidays, employees, workInfo, leaveRequests,
		employees, companies, departments, currentGradeSelect)

	// Build dynamic parameters list
	params := []interface{}{startDate, endDate}
	paramCounter := 3

	// Add company filter if provided
	if companyID != nil {
		query += fmt.Sprintf("\n\t\t\tAND c.id = $%d", paramCounter)
		params = append(params, *companyID)
		paramCounter++
	}

	// Add department filter with IN operator if provided
	if len(departmentIDs) > 0 {
		placeholders := make([]string, 0, len(departmentIDs))
		for _, departmentID := range departmentIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramCounter))
			params = append(params, departmentID)
			paramCounter++
		}

		query += fmt.Sprintf("\n\t\t\tAND d.id IN (%s)", strings.Join(placeholders, ", "))
	}

	// Exclude interns (job_position_id = 21)
	query += fmt.Sprintf("\n\t\t\tAND NOT EXISTS (SELECT 1 FROM %s wi2 WHERE wi2.employee_id = e.id AND wi2.job_position_id = 21 AND wi2.deleted = false AND wi2.start_date <= $1::date AND (wi2.end_date IS NULL OR wi2.end_date >= $1::date))", workInfo)

	// Add active/passive filter
	if isActive != nil {
		if *isActive {
			query += "\n\t\t\tAND e.leave_date IS NULL"
		} else {
			query += "\n\t\t\tAND e.leave_date IS NOT NULL"
		}
	} else {
		// Default: only active employees
		query += "\n\t\t\tAND e.leave_date IS NULL"
	}

	// Add GROUP BY and ORDER BY
	query += `
		GROUP BY
			e.id,
			e.identity_no,
			cg.current_grade,
			e.first_name,
			e.last_name,
			c.id,
			c.name,
			d.id,
			d.name,
			d.manager,
			e.hire_date,
			e.leave_date
		ORDER BY e.id
	`

	err := r.db.Raw(query, params...).Scan(&rows).Error
	return rows, err
}

// GetGradeReportData executes the grade report SQL query.
// current_grade is ACTIVE EmployeeGrade SoT (not date-window history).
// total_gap / experience formulas are unchanged.
func (r *employeeRepository) GetGradeReportData(companyID *uint, departmentIDs []uint, isActive *bool) ([]types.GradeReportRow, error) {
	var rows []types.GradeReportRow

	employees := domain.GetTableName("hr_employees")
	grades := domain.GetTableName("hr_grades")
	workInfo := domain.GetTableName("hr_employee_work_information")
	companies := domain.GetTableName("hr_companies")
	departments := domain.GetTableName("hr_departments")
	currentGradeSelect := buildActiveCurrentGradeSelectSQL()

	query := fmt.Sprintf(`
WITH experience_calc AS (
    SELECT
        e.id,
        e.first_name,
        e.last_name,
        e.hire_date,
        e.leave_date,
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


    FROM %s e
    WHERE e.deleted = false
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
    LEFT JOIN %s g
        ON egc.expected_experience >= g.min_year
            AND egc.expected_experience < g.max_year
),

current_grade AS (
%s
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
    FROM %s wi
    LEFT JOIN %s c
        ON c.id = wi.company_id
    LEFT JOIN %s d
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
WHERE 1=1`,
		employees, grades, currentGradeSelect, workInfo, companies, departments)

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
	if len(departmentIDs) > 0 {
		placeholders := make([]string, 0, len(departmentIDs))
		for _, depID := range departmentIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramCounter))
			params = append(params, depID)
			paramCounter++
		}
		query += fmt.Sprintf("\n  AND t.department_id IN (%s)", strings.Join(placeholders, ", "))
	}

	// Exclude interns (job_position_id = 21)
	query += fmt.Sprintf(`
  AND NOT EXISTS (
    SELECT 1 FROM %s wi2
    WHERE wi2.employee_id = e.id
      AND wi2.job_position_id = 21
      AND wi2.deleted = false
      AND wi2.start_date <= CURRENT_DATE
      AND (wi2.end_date IS NULL OR wi2.end_date >= CURRENT_DATE)
  )`, workInfo)

	// Add active/passive filter
	if isActive != nil {
		if *isActive {
			query += "\n  AND e.leave_date IS NULL"
		} else {
			query += "\n  AND e.leave_date IS NOT NULL"
		}
	} else {
		// Default: only active employees
		query += "\n  AND e.leave_date IS NULL"
	}

	// Complete the query
	query += `
ORDER BY e.id`

	err := r.db.Raw(query, params...).Scan(&rows).Error
	return rows, err
}

// GetContractReportData executes the contract report SQL query
func (r *employeeRepository) GetContractReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.ContractReportRow, error) {
	var rows []types.ContractReportRow

	query := `
		SELECT
			e.id,
			e.first_name,
			e.last_name,
			c.name AS company_name,
			d.name AS department_name,
			d.manager AS manager,
			COALESCE(string_agg(ctr.project_name || ' (' || ctr.contract_no || ')', ', '), '') AS contract_names
		FROM hr_employees e
		JOIN hr_employee_work_information wi ON wi.employee_id = e.id AND wi.deleted = false
			AND wi.id = (
				SELECT id FROM hr_employee_work_information
				WHERE employee_id = e.id AND deleted = false
				ORDER BY start_date DESC LIMIT 1
			)
		LEFT JOIN hr_companies c ON c.id = wi.company_id AND c.deleted = false
		LEFT JOIN hr_departments d ON d.id = wi.department_id AND d.deleted = false
		LEFT JOIN hr_employee_contracts ec ON ec.employee_id = e.id AND ec.deleted = false
		LEFT JOIN hr_contracts ctr ON ctr.id = ec.contract_id AND ctr.deleted = false`

	var params []interface{}

	if startDate != "" && endDate != "" {
		query += " AND ctr.start_date <= ?::date AND (ctr.end_date IS NULL OR ctr.end_date >= ?::date)"
		params = append(params, endDate, startDate)
	}

	query += `
		WHERE e.deleted = false`

	if companyID != nil {
		query += " AND wi.company_id = ?"
		params = append(params, *companyID)
	}

	if len(departmentIDs) > 0 {
		query += " AND wi.department_id IN ?"
		params = append(params, departmentIDs)
	}

	if isActive != nil {
		if *isActive {
			query += " AND e.leave_date IS NULL"
		} else {
			query += " AND e.leave_date IS NOT NULL"
		}
	} else {
		// Default to active
		query += " AND e.leave_date IS NULL"
	}

	query += " GROUP BY e.id, c.name, d.name, d.manager ORDER BY e.id"

	err := r.db.Raw(query, params...).Scan(&rows).Error
	return rows, err
}
