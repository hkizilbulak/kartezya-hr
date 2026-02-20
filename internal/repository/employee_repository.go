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
	GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.Employee, int64, error)
	Update(employee *domain.Employee, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	GetTotalCount() (int64, error)
	GetTotalCountWithFilters(filters map[string]interface{}) (int64, error)
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

// GetAllWithFilters returns filtered employees with pagination and sorting
func (r *employeeRepository) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.Employee, int64, error) {
	var employees []*domain.Employee
	var total int64

	// Replace hardcoded table names with dynamic table names using domain.GetTableName
	query := r.db.Model(&domain.Employee{}).Where(fmt.Sprintf("%s.deleted = ?", domain.GetTableName("hr_employees")), false)

	// Apply filters
	if filters != nil {
		if id, ok := filters["id"]; ok {
			query = query.Where(fmt.Sprintf("%s.id = ?", domain.GetTableName("hr_employees")), id)
		}

		if firstName, ok := filters["first_name"]; ok {
			query = query.Where(fmt.Sprintf("LOWER(%s.first_name) LIKE LOWER(?)", domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", firstName)+"%")
		}

		if email, ok := filters["email"]; ok {
			emailStr := fmt.Sprintf("%v", email)
			query = query.Where(fmt.Sprintf("LOWER(%s.email) LIKE LOWER(?) OR LOWER(%s.company_email) LIKE LOWER(?)", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), "%"+emailStr+"%", "%"+emailStr+"%")
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

		// Department IDs filter - only if company_id is provided (since we need the JOINs)
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

		// Manager filter - needs department JOIN
		if manager, ok := filters["manager"]; ok {
			// If we haven't added JOINs yet, add them now
			if _, hasCompany := filters["company_id"]; !hasCompany {
				if _, hasDepartmentIDs := filters["department_ids"]; !hasDepartmentIDs {
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
			}
			query = query.Where(fmt.Sprintf("LOWER(%s.manager) LIKE LOWER(?)", domain.GetTableName("hr_departments")), "%"+fmt.Sprintf("%v", manager)+"%")
		}

		if status, ok := filters["status"]; ok {
			query = query.Where(fmt.Sprintf("%s.status = ?", domain.GetTableName("hr_employees")), status)
		}

		if identityNo, ok := filters["identity_no"]; ok {
			query = query.Where(fmt.Sprintf("%s.identity_no = ?", domain.GetTableName("hr_employees")), identityNo)
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
		if id, ok := filters["id"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("%s.id = ?", domain.GetTableName("hr_employees")), id)
		}
		if firstName, ok := filters["first_name"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.first_name) LIKE LOWER(?)", domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", firstName)+"%")
		}
		if email, ok := filters["email"]; ok {
			emailStr := fmt.Sprintf("%v", email)
			countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.email) LIKE LOWER(?) OR LOWER(%s.company_email) LIKE LOWER(?)", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), "%"+emailStr+"%", "%"+emailStr+"%")
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
			countQuery = countQuery.Where(fmt.Sprintf("LOWER(%s.manager) LIKE LOWER(?)", domain.GetTableName("hr_departments")), "%"+fmt.Sprintf("%v", manager)+"%")
		}
		if status, ok := filters["status"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("%s.status = ?", domain.GetTableName("hr_employees")), status)
		}
		if identityNo, ok := filters["identity_no"]; ok {
			countQuery = countQuery.Where(fmt.Sprintf("%s.identity_no = ?", domain.GetTableName("hr_employees")), identityNo)
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
		// ID filter
		if id, ok := filters["id"]; ok {
			query = query.Where(fmt.Sprintf("%s.id = ?", domain.GetTableName("hr_employees")), id)
		}

		// First name filter (LIKE search)
		if firstName, ok := filters["first_name"]; ok {
			query = query.Where(fmt.Sprintf("LOWER(%s.first_name) LIKE LOWER(?)", domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", firstName)+"%")
		}

		// Email filter (LIKE search for both personal and company email)
		if email, ok := filters["email"]; ok {
			emailStr := fmt.Sprintf("%v", email)
			query = query.Where(fmt.Sprintf("LOWER(%s.email) LIKE LOWER(?) OR LOWER(%s.company_email) LIKE LOWER(?)", domain.GetTableName("hr_employees"), domain.GetTableName("hr_employees")), "%"+emailStr+"%", "%"+emailStr+"%")
		}

		// Identity number filter
		if identityNo, ok := filters["identity_no"]; ok {
			query = query.Where(fmt.Sprintf("%s.identity_no LIKE ?", domain.GetTableName("hr_employees")), "%"+fmt.Sprintf("%v", identityNo)+"%")
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
			query = query.Where(fmt.Sprintf("LOWER(%s.manager) LIKE LOWER(?)", domain.GetTableName("hr_departments")), "%"+fmt.Sprintf("%v", manager)+"%")
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
		"is_grade_up":                employee.IsGradeUp,
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
		Joins(fmt.Sprintf("JOIN %s ON %s.employee_id = %s.id",
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employees"))).
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.job_position_id",
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_job_positions"),
			domain.GetTableName("hr_employee_work_information"))).
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
		Joins(fmt.Sprintf("JOIN %s ON %s.employee_id = %s.id",
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employee_work_information"),
			domain.GetTableName("hr_employees"))).
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.department_id",
			domain.GetTableName("hr_departments"),
			domain.GetTableName("hr_departments"),
			domain.GetTableName("hr_employee_work_information"))).
		Joins(fmt.Sprintf("JOIN %s ON %s.id = %s.company_id",
			domain.GetTableName("hr_companies"),
			domain.GetTableName("hr_companies"),
			domain.GetTableName("hr_departments"))).
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
