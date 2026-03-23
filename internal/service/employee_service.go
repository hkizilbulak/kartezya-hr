package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type EmployeeService interface {
	CreateEmployee(email, companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalGap float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation string, gradeID *int64, contractNo, professionStartDate, note, motherName, fatherName, nationality, identityNo string, createdBy string, roles []string) (*domain.Employee, error)
	GetEmployeeByID(id uint) (*types.EmployeeDetailResponse, error)
	GetEmployeeByUserID(userID uint) (*types.EmployeeDetailResponse, error)
	UpdateEmployee(id uint, email, companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalGap float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation string, gradeID *int64, contractNo, professionStartDate, note, motherName, fatherName, nationality, identityNo, status string, modifiedBy string, requestingUserID uint, isAdmin bool, roles []string) error
	UpdateMyProfile(userID uint, email, phone, address, state, city, gender, dateOfBirth string, professionStartDate string, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation, motherName, fatherName, nationality, identityNo string) error
	DeleteEmployee(id uint, deletedBy string, isAdmin bool) error
	ListEmployees(limit, offset int, isAdmin bool) ([]*types.EmployeeResponse, error)
	ListEmployeesWithFilters(limit, offset int, sortField, sortDirection string, filters map[string]interface{}, isAdmin bool) ([]*types.EmployeeResponse, error)
	GetTotalCount() (int64, error)
	GetTotalCountWithFilters(filters map[string]interface{}) (int64, error)
	GetEmployeeCountByGender() ([]interface{}, error)
	GetEmployeeCountByPosition() ([]interface{}, error)
	GetEmployeeCountByCompanyDepartment() ([]interface{}, error)
	GetEmployeeCountByGrade() ([]interface{}, error)
}

type employeeService struct {
	employeeRepo repository.EmployeeRepository
	userRepo     repository.UserRepository
	userRoleRepo repository.UserRoleRepository
	roleRepo     repository.RoleRepository
	authService  AuthService
	auditService AuditService
	workInfoRepo repository.WorkInformationRepository
	emailService EmailService
}

func NewEmployeeService(employeeRepo repository.EmployeeRepository, userRepo repository.UserRepository, userRoleRepo repository.UserRoleRepository, roleRepo repository.RoleRepository, authService AuthService, auditService AuditService, workInfoRepo repository.WorkInformationRepository, emailService EmailService) EmployeeService {
	return &employeeService{
		employeeRepo: employeeRepo,
		userRepo:     userRepo,
		userRoleRepo: userRoleRepo,
		roleRepo:     roleRepo,
		authService:  authService,
		auditService: auditService,
		workInfoRepo: workInfoRepo,
		emailService: emailService,
	}
}

func (s *employeeService) CreateEmployee(email, companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalGap float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation string, gradeID *int64, contractNo, professionStartDate, note, motherName, fatherName, nationality, identityNo string, createdBy string, roles []string) (*domain.Employee, error) {
	// Delegate user creation to AuthService (respecting domain boundaries)
	// companyEmail is used as the user's email in the user table
	user, err := s.authService.CreateUserForEmployee(companyEmail, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Parse date fields using the new parseDate function
	dateOfBirthPtr, _ := parseDate(dateOfBirth)
	hireDatePtr, _ := parseDate(hireDate)
	leaveDatePtr, _ := parseDate(leaveDate)
	professionStartDatePtr, _ := parseDate(professionStartDate)

	// Create employee profile
	// email is stored in employee.email (personal email)
	// companyEmail is stored in employee.company_email (corporate email)
	employee := &domain.Employee{
		UserID:                   user.ID,
		Email:                    email,
		CompanyEmail:             companyEmail,
		FirstName:                firstName,
		LastName:                 lastName,
		Phone:                    phone,
		Address:                  address,
		State:                    state,
		City:                     city,
		Gender:                   gender,
		DateOfBirth:              dateOfBirthPtr,
		HireDate:                 hireDatePtr,
		LeaveDate:                leaveDatePtr,
		TotalGap:                 totalGap,
		MaritalStatus:            maritalStatus,
		EmergencyContact:         emergencyContact,
		EmergencyContactName:     emergencyContactName,
		EmergencyContactRelation: emergencyContactRelation,
		GradeID:                  gradeID,
		ContractNo:               contractNo,
		ProfessionStartDate:      professionStartDatePtr,
		Note:                     note,
		MotherName:               motherName,
		FatherName:               fatherName,
		Nationality:              nationality,
		IdentityNo:               identityNo,
	}

	// Check if an employee with the same company email already exists and is active
	existingEmployee, err := s.employeeRepo.GetByCompanyEmail(companyEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to check company email: %w", err)
	}
	if existingEmployee != nil {
		return nil, fmt.Errorf("Bu şirket e-posta adresine sahip aktif çalışan var")
	}

	// Check if an employee with the same personal email already exists and is active
	if email != "" {
		existingEmailEmployee, err := s.employeeRepo.GetByEmail(email)
		if err != nil {
			return nil, fmt.Errorf("failed to check personal email: %w", err)
		}
		if existingEmailEmployee != nil {
			return nil, fmt.Errorf("Bu kişisel e-posta adresine sahip aktif çalışan var")
		}
	}

	// Check if an employee with the same identity number already exists and is active
	if identityNo != "" {
		existingIdentityEmployee, err := s.employeeRepo.GetByIdentityNo(identityNo)
		if err != nil {
			return nil, fmt.Errorf("failed to check identity number: %w", err)
		}
		if existingIdentityEmployee != nil {
			return nil, fmt.Errorf("Bu kimlik numarasına sahip aktif çalışan var")
		}
	}

	// Check if an employee with the same phone number already exists and is active
	if phone != "" {
		existingPhoneEmployee, err := s.employeeRepo.GetByPhone(phone)
		if err != nil {
			return nil, fmt.Errorf("failed to check phone number: %w", err)
		}
		if existingPhoneEmployee != nil {
			return nil, fmt.Errorf("Bu telefon numarasına sahip aktif çalışan var")
		}
	}

	// Create the employee
	if err := s.employeeRepo.Create(employee, createdBy); err != nil {
		return nil, fmt.Errorf("failed to create employee: %w", err)
	}

	fmt.Printf("Employee created successfully. User ID: %d, Employee ID: %d\n", user.ID, employee.ID)

	// Assign roles to user if provided
	if len(roles) > 0 {
		fmt.Printf("Attempting to assign %d roles to user %d\n", len(roles), user.ID)
		if err := s.assignRolesToUser(user.ID, roles, createdBy); err != nil {
			// Log error but don't fail the operation - employee is already created
			fmt.Printf("ERROR: failed to assign roles to user %d: %v\n", user.ID, err)
		}
	} else {
		fmt.Printf("No roles provided for user %d\n", user.ID)
	}

	// Send password reset email to the newly created user
	fmt.Printf("Sending password reset email to: %s\n", companyEmail)
	if err := s.emailService.SendPasswordResetEmail(user, firstName, lastName); err != nil {
		// Log error but don't fail the operation - employee is already created
		fmt.Printf("WARNING: failed to send password reset email to %s: %v\n", companyEmail, err)
	} else {
		fmt.Printf("Password reset email sent successfully to: %s\n", companyEmail)
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("Employee", employee.ID, domain.AuditActionCreate, nil, employee, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return employee, nil
}

func (s *employeeService) GetEmployeeByID(id uint) (*types.EmployeeDetailResponse, error) {
	employee, err := s.employeeRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(employee.UserID)
	if err != nil {
		return nil, err
	}

	// Get user roles
	userRoles, err := s.userRoleRepo.GetRolesByUserID(user.ID)
	if err != nil {
		fmt.Printf("Warning: failed to get roles for user %d: %v\n", user.ID, err)
	}

	roleNames := make([]string, len(userRoles))
	for i, role := range userRoles {
		roleNames[i] = role.Name
	}

	// Get work information list (sorted by start_date DESC - newest first)
	var workInfoList []types.EmployeeWorkInformationList
	workInfos, err := s.workInfoRepo.GetByEmployeeID(employee.ID)
	if err == nil && len(workInfos) > 0 {
		// Sort by start_date DESC (newest first)
		workInfoList = make([]types.EmployeeWorkInformationList, len(workInfos))
		for i, workInfo := range workInfos {
			endDateStr := (*string)(nil)
			if workInfo.EndDate != nil {
				endStr := workInfo.EndDate.Format(time.RFC3339)
				endDateStr = &endStr
			}

			workInfoList[i] = types.EmployeeWorkInformationList{
				ID:             workInfo.ID,
				CompanyName:    workInfo.Company.Name,
				DepartmentName: workInfo.Department.Name,
				Manager:        workInfo.Department.Manager,
				JobTitle:       workInfo.JobPosition.Title,
				StartDate:      workInfo.StartDate.Format(time.RFC3339),
				EndDate:        endDateStr,
			}
		}
		// Reverse to get DESC order
		for i, j := 0, len(workInfoList)-1; i < j; i, j = i+1, j-1 {
			workInfoList[i], workInfoList[j] = workInfoList[j], workInfoList[i]
		}
	}

	// Convert dates to string format for JSON response
	var dateOfBirthStr, hireDateStr, leaveDateStr, professionStartDateStr *string

	if employee.DateOfBirth != nil {
		dateStr := employee.DateOfBirth.Format(time.RFC3339)
		dateOfBirthStr = &dateStr
	}

	if employee.HireDate != nil {
		hireStr := employee.HireDate.Format(time.RFC3339)
		hireDateStr = &hireStr
	}

	if employee.LeaveDate != nil {
		leaveStr := employee.LeaveDate.Format(time.RFC3339)
		leaveDateStr = &leaveStr
	}

	if employee.ProfessionStartDate != nil {
		profStr := employee.ProfessionStartDate.Format(time.RFC3339)
		professionStartDateStr = &profStr
	}

	userInfo := types.UserInfo{
		ID:    user.ID,
		Email: user.Email,
	}

	return &types.EmployeeDetailResponse{
		ID:                       employee.ID,
		User:                     userInfo,
		FirstName:                employee.FirstName,
		LastName:                 employee.LastName,
		Email:                    employee.Email,
		CompanyEmail:             employee.CompanyEmail,
		Phone:                    employee.Phone,
		Address:                  employee.Address,
		State:                    employee.State,
		City:                     employee.City,
		Gender:                   employee.Gender,
		DateOfBirth:              dateOfBirthStr,
		HireDate:                 hireDateStr,
		LeaveDate:                leaveDateStr,
		TotalGap:                 employee.TotalGap,
		MaritalStatus:            employee.MaritalStatus,
		EmergencyContact:         employee.EmergencyContact,
		EmergencyContactName:     employee.EmergencyContactName,
		EmergencyContactRelation: employee.EmergencyContactRelation,
		GradeID:                  employee.GradeID,
		ContractNo:               employee.ContractNo,
		ProfessionStartDate:      professionStartDateStr,
		Note:                     employee.Note,
		MotherName:               employee.MotherName,
		FatherName:               employee.FatherName,
		Nationality:              employee.Nationality,
		IdentityNo:               employee.IdentityNo,
		Roles:                    roleNames,
		WorkInformation:          workInfoList,
		Status:                   employee.Status,
	}, nil
}

func (s *employeeService) GetEmployeeByUserID(userID uint) (*types.EmployeeDetailResponse, error) {
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(employee.UserID)
	if err != nil {
		return nil, err
	}

	// Get user roles
	userRoles, err := s.userRoleRepo.GetRolesByUserID(user.ID)
	if err != nil {
		fmt.Printf("Warning: failed to get roles for user %d: %v\n", user.ID, err)
	}

	roleNames := make([]string, len(userRoles))
	for i, role := range userRoles {
		roleNames[i] = role.Name
	}

	// Get work information list (sorted by start_date DESC - newest first)
	var workInfoList []types.EmployeeWorkInformationList
	workInfos, err := s.workInfoRepo.GetByEmployeeID(employee.ID)
	if err == nil && len(workInfos) > 0 {
		// Sort by start_date DESC (newest first)
		workInfoList = make([]types.EmployeeWorkInformationList, len(workInfos))
		for i, workInfo := range workInfos {
			endDateStr := (*string)(nil)
			if workInfo.EndDate != nil {
				endStr := workInfo.EndDate.Format(time.RFC3339)
				endDateStr = &endStr
			}

			workInfoList[i] = types.EmployeeWorkInformationList{
				ID:             workInfo.ID,
				CompanyName:    workInfo.Company.Name,
				DepartmentName: workInfo.Department.Name,
				Manager:        workInfo.Department.Manager,
				JobTitle:       workInfo.JobPosition.Title,
				StartDate:      workInfo.StartDate.Format(time.RFC3339),
				EndDate:        endDateStr,
			}
		}
	}

	// Convert dates to string format for JSON response
	var dateOfBirthStr, hireDateStr, leaveDateStr, professionStartDateStr *string

	if employee.DateOfBirth != nil {
		dateStr := employee.DateOfBirth.Format(time.RFC3339)
		dateOfBirthStr = &dateStr
	}

	if employee.HireDate != nil {
		hireStr := employee.HireDate.Format(time.RFC3339)
		hireDateStr = &hireStr
	}

	if employee.LeaveDate != nil {
		leaveStr := employee.LeaveDate.Format(time.RFC3339)
		leaveDateStr = &leaveStr
	}

	if employee.ProfessionStartDate != nil {
		profStr := employee.ProfessionStartDate.Format(time.RFC3339)
		professionStartDateStr = &profStr
	}

	userInfo := types.UserInfo{
		ID:    user.ID,
		Email: user.Email,
	}

	return &types.EmployeeDetailResponse{
		ID:                       employee.ID,
		User:                     userInfo,
		FirstName:                employee.FirstName,
		LastName:                 employee.LastName,
		Email:                    employee.Email,
		CompanyEmail:             employee.CompanyEmail,
		Phone:                    employee.Phone,
		Address:                  employee.Address,
		State:                    employee.State,
		City:                     employee.City,
		Gender:                   employee.Gender,
		DateOfBirth:              dateOfBirthStr,
		HireDate:                 hireDateStr,
		LeaveDate:                leaveDateStr,
		TotalGap:                 employee.TotalGap,
		MaritalStatus:            employee.MaritalStatus,
		EmergencyContact:         employee.EmergencyContact,
		EmergencyContactName:     employee.EmergencyContactName,
		EmergencyContactRelation: employee.EmergencyContactRelation,
		GradeID:                  employee.GradeID,
		ContractNo:               employee.ContractNo,
		ProfessionStartDate:      professionStartDateStr,
		Note:                     employee.Note,
		MotherName:               employee.MotherName,
		FatherName:               employee.FatherName,
		Nationality:              employee.Nationality,
		IdentityNo:               employee.IdentityNo,
		Roles:                    roleNames,
		WorkInformation:          workInfoList,
		Status:                   employee.Status,
	}, nil
}

func (s *employeeService) UpdateEmployee(id uint, email, companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalGap float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation string, gradeID *int64, contractNo, professionStartDate, note, motherName, fatherName, nationality, identityNo, status string, modifiedBy string, requestingUserID uint, isAdmin bool, roles []string) error {
	// Get existing employee for authorization check and audit trail
	existingEmployee, err := s.employeeRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check authorization - employees can only update their own profile, admins can update any
	if !isAdmin && existingEmployee.UserID != requestingUserID {
		return errors.New("unauthorized to update this employee profile")
	}

	// Update user email if it has changed
	existingUser, err := s.userRepo.GetByID(existingEmployee.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	if existingUser.Email != companyEmail {
		// Check if new email already exists for another user
		if existingEmailUser, err := s.userRepo.GetByEmail(companyEmail); err == nil && existingEmailUser.ID != existingUser.ID {
			return fmt.Errorf("email %s is already in use by another user", companyEmail)
		}

		// Update user email
		existingUser.Email = companyEmail
		if err := s.userRepo.Update(existingUser, modifiedBy); err != nil {
			return fmt.Errorf("failed to update user email: %v", err)
		}
	}

	// Check if company email has changed and if new email already exists
	if existingEmployee.CompanyEmail != companyEmail {
		existingEmailEmployee, err := s.employeeRepo.GetByCompanyEmail(companyEmail)
		if err != nil {
			return fmt.Errorf("failed to check company email: %w", err)
		}
		if existingEmailEmployee != nil && existingEmailEmployee.ID != id {
			return fmt.Errorf("Bu şirket e-posta adresine sahip aktif çalışan var")
		}
	}

	// Check if identity number has changed and if new identity number already exists
	if existingEmployee.IdentityNo != identityNo && identityNo != "" {
		existingIdentityEmployee, err := s.employeeRepo.GetByIdentityNo(identityNo)
		if err != nil {
			return fmt.Errorf("failed to check identity number: %w", err)
		}
		if existingIdentityEmployee != nil && existingIdentityEmployee.ID != id {
			return fmt.Errorf("Bu kimlik numarasına sahip aktif çalışan var")
		}
	}

	// Check if phone number has changed and if new phone number already exists
	if existingEmployee.Phone != phone && phone != "" {
		existingPhoneEmployee, err := s.employeeRepo.GetByPhone(phone)
		if err != nil {
			return fmt.Errorf("failed to check phone number: %w", err)
		}
		if existingPhoneEmployee != nil && existingPhoneEmployee.ID != id {
			return fmt.Errorf("Bu telefon numarasına sahip aktif çalışan var")
		}
	}

	// Check if company email has changed and if new email already exists
	if existingEmployee.CompanyEmail != companyEmail {
		existingEmailEmployee, err := s.employeeRepo.GetByCompanyEmail(companyEmail)
		if err != nil {
			return fmt.Errorf("failed to check company email: %w", err)
		}
		if existingEmailEmployee != nil && existingEmailEmployee.ID != id {
			return fmt.Errorf("Bu şirket e-posta adresine sahip aktif çalışan var")
		}
	}

	// Check if identity number has changed and if new identity number already exists
	if existingEmployee.IdentityNo != identityNo && identityNo != "" {
		existingIdentityEmployee, err := s.employeeRepo.GetByIdentityNo(identityNo)
		if err != nil {
			return fmt.Errorf("failed to check identity number: %w", err)
		}
		if existingIdentityEmployee != nil && existingIdentityEmployee.ID != id {
			return fmt.Errorf("Bu kimlik numarasına sahip aktif çalışan var")
		}
	}

	// Check if phone number has changed and if new phone number already exists
	if existingEmployee.Phone != phone && phone != "" {
		existingPhoneEmployee, err := s.employeeRepo.GetByPhone(phone)
		if err != nil {
			return fmt.Errorf("failed to check phone number: %w", err)
		}
		if existingPhoneEmployee != nil && existingPhoneEmployee.ID != id {
			return fmt.Errorf("Bu telefon numarasına sahip aktif çalışan var")
		}
	}

	// Parse date fields using the parseDate helper function
	dateOfBirthPtr, _ := parseDate(dateOfBirth)
	hireDatePtr, _ := parseDate(hireDate)
	leaveDatePtr, _ := parseDate(leaveDate)
	professionStartDatePtr, _ := parseDate(professionStartDate)

	// Clone the existing employee and update only the provided fields
	updatedEmployee := *existingEmployee
	updatedEmployee.Email = email
	updatedEmployee.CompanyEmail = companyEmail
	updatedEmployee.FirstName = firstName
	updatedEmployee.LastName = lastName
	updatedEmployee.Phone = phone
	updatedEmployee.Address = address
	updatedEmployee.State = state
	updatedEmployee.City = city
	updatedEmployee.Gender = gender
	updatedEmployee.DateOfBirth = dateOfBirthPtr
	updatedEmployee.HireDate = hireDatePtr
	updatedEmployee.LeaveDate = leaveDatePtr
	updatedEmployee.TotalGap = totalGap
	updatedEmployee.MaritalStatus = maritalStatus
	updatedEmployee.EmergencyContact = emergencyContact
	updatedEmployee.EmergencyContactName = emergencyContactName
	updatedEmployee.EmergencyContactRelation = emergencyContactRelation
	updatedEmployee.GradeID = gradeID
	updatedEmployee.ContractNo = contractNo
	updatedEmployee.ProfessionStartDate = professionStartDatePtr
	updatedEmployee.Note = note
	updatedEmployee.MotherName = motherName
	updatedEmployee.FatherName = fatherName
	updatedEmployee.Nationality = nationality
	updatedEmployee.IdentityNo = identityNo
	updatedEmployee.Status = status

	// Update employee
	if err := s.employeeRepo.Update(&updatedEmployee, modifiedBy); err != nil {
		return err
	}

	// Update roles if provided
	if len(roles) > 0 {
		if err := s.updateUserRoles(existingEmployee.UserID, roles, modifiedBy); err != nil {
			// Log error but don't fail the operation - employee is already updated
			fmt.Printf("Warning: failed to update roles for user %d: %v\n", existingEmployee.UserID, err)
		}
	}

	// Get updated employee for audit
	auditedEmployee, _ := s.employeeRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("Employee", id, domain.AuditActionUpdate, existingEmployee, auditedEmployee, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *employeeService) UpdateMyProfile(userID uint, email, phone, address, state, city, gender, dateOfBirth string, professionStartDate string, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation, motherName, fatherName, nationality, identityNo string) error {
	// Get employee by user ID
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("employee not found: %v", err)
	}

	// Parse date fields using the new parseDate function
	dateOfBirthPtr, _ := parseDate(dateOfBirth)
	professionStartDatePtr, _ := parseDate(professionStartDate)

	// Clone the existing employee and update all fields (no empty checks)
	updatedEmployee := *employee
	updatedEmployee.Email = email
	updatedEmployee.Phone = phone
	updatedEmployee.Address = address
	updatedEmployee.State = state
	updatedEmployee.City = city
	updatedEmployee.Gender = gender
	updatedEmployee.DateOfBirth = dateOfBirthPtr
	updatedEmployee.MaritalStatus = maritalStatus
	updatedEmployee.EmergencyContact = emergencyContact
	updatedEmployee.EmergencyContactName = emergencyContactName
	updatedEmployee.EmergencyContactRelation = emergencyContactRelation
	updatedEmployee.MotherName = motherName
	updatedEmployee.FatherName = fatherName
	updatedEmployee.Nationality = nationality
	updatedEmployee.IdentityNo = identityNo
	updatedEmployee.ProfessionStartDate = professionStartDatePtr

	// Update employee
	if err := s.employeeRepo.Update(&updatedEmployee, ""); err != nil {
		return fmt.Errorf("failed to update profile: %v", err)
	}

	// Audit the update
	if err := s.auditService.CreateAuditLog("Employee", employee.ID, domain.AuditActionUpdate, employee, &updatedEmployee, ""); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *employeeService) DeleteEmployee(id uint, deletedBy string, isAdmin bool) error {
	if !isAdmin {
		return errors.New("only administrators can delete employee profiles")
	}

	// Get existing employee for audit trail
	existingEmployee, err := s.employeeRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the employee
	if err := s.employeeRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("Employee", id, domain.AuditActionDelete, existingEmployee, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *employeeService) ListEmployees(limit, offset int, isAdmin bool) ([]*types.EmployeeResponse, error) {
	if !isAdmin {
		return nil, errors.New("only administrators can list all employees")
	}

	sortParams := types.SortParams{
		Sort:      "created_at",
		Direction: "DESC",
	}

	employees, _, err := s.employeeRepo.GetAll(limit, offset, sortParams)
	if err != nil {
		return nil, err
	}

	// Convert domain.Employee to EmployeeResponse with roles and work information
	responses := make([]*types.EmployeeResponse, len(employees))
	for i, employee := range employees {
		user, err := s.userRepo.GetByID(employee.UserID)
		if err != nil {
			fmt.Printf("Warning: failed to get user %d: %v\n", employee.UserID, err)
			continue
		}

		// Get user roles
		userRoles, err := s.userRoleRepo.GetRolesByUserID(user.ID)
		if err != nil {
			fmt.Printf("Warning: failed to get roles for user %d: %v\n", user.ID, err)
		}

		roleNames := make([]string, len(userRoles))
		for j, role := range userRoles {
			roleNames[j] = role.Name
		}

		// Get work information - get latest (first one in DESC order)
		var workInfoLookup *types.EmployeeWorkInfoLookup
		workInfos, err := s.workInfoRepo.GetByEmployeeID(employee.ID)
		if err == nil && len(workInfos) > 0 {
			// Get the latest work information (first one - already DESC sorted by database)
			latestWorkInfo := workInfos[0]
			workInfoLookup = &types.EmployeeWorkInfoLookup{
				CompanyName:    latestWorkInfo.Company.Name,
				DepartmentName: latestWorkInfo.Department.Name,
				Manager:        latestWorkInfo.Department.Manager,
				JobTitle:       latestWorkInfo.JobPosition.Title,
			}
		}

		// Convert dates to string format for JSON response
		var dateOfBirthStr, hireDateStr, leaveDateStr, professionStartDateStr *string

		if employee.DateOfBirth != nil {
			dateStr := employee.DateOfBirth.Format(time.RFC3339)
			dateOfBirthStr = &dateStr
		}

		if employee.HireDate != nil {
			hireStr := employee.HireDate.Format(time.RFC3339)
			hireDateStr = &hireStr
		}

		if employee.LeaveDate != nil {
			leaveStr := employee.LeaveDate.Format(time.RFC3339)
			leaveDateStr = &leaveStr
		}

		if employee.ProfessionStartDate != nil {
			profStr := employee.ProfessionStartDate.Format(time.RFC3339)
			professionStartDateStr = &profStr
		}

		userInfo := types.UserInfo{
			ID:    user.ID,
			Email: user.Email,
		}

		responses[i] = &types.EmployeeResponse{
			ID:                       employee.ID,
			User:                     userInfo,
			FirstName:                employee.FirstName,
			LastName:                 employee.LastName,
			Email:                    employee.Email,
			CompanyEmail:             employee.CompanyEmail,
			Phone:                    employee.Phone,
			Address:                  employee.Address,
			State:                    employee.State,
			City:                     employee.City,
			Gender:                   employee.Gender,
			DateOfBirth:              dateOfBirthStr,
			HireDate:                 hireDateStr,
			LeaveDate:                leaveDateStr,
			TotalGap:                 employee.TotalGap,
			MaritalStatus:            employee.MaritalStatus,
			EmergencyContact:         employee.EmergencyContact,
			EmergencyContactName:     employee.EmergencyContactName,
			EmergencyContactRelation: employee.EmergencyContactRelation,
			GradeID:                  employee.GradeID,
			ContractNo:               employee.ContractNo,
			ProfessionStartDate:      professionStartDateStr,
			Note:                     employee.Note,
			MotherName:               employee.MotherName,
			FatherName:               employee.FatherName,
			Nationality:              employee.Nationality,
			IdentityNo:               employee.IdentityNo,
			Roles:                    roleNames,
			WorkInformation:          workInfoLookup,
		}
	}

	return responses, nil
}

// ListEmployeesWithFilters returns filtered employees with pagination and sorting
func (s *employeeService) ListEmployeesWithFilters(limit, offset int, sortField, sortDirection string, filters map[string]interface{}, isAdmin bool) ([]*types.EmployeeResponse, error) {
	if !isAdmin {
		return nil, errors.New("only administrators can list all employees")
	}

	sortParams := types.SortParams{
		Sort:      sortField,
		Direction: sortDirection,
	}

	employees, _, err := s.employeeRepo.GetAllWithFilters(limit, offset, sortParams, filters)
	if err != nil {
		return nil, err
	}

	// Convert domain.Employee to EmployeeResponse with roles and work information
	responses := make([]*types.EmployeeResponse, len(employees))
	for i, employee := range employees {
		user, err := s.userRepo.GetByID(employee.UserID)
		if err != nil {
			fmt.Printf("Warning: failed to get user %d: %v\n", employee.UserID, err)
			continue
		}

		// Get user roles
		userRoles, err := s.userRoleRepo.GetRolesByUserID(user.ID)
		if err != nil {
			fmt.Printf("Warning: failed to get roles for user %d: %v\n", user.ID, err)
		}

		roleNames := make([]string, len(userRoles))
		for j, role := range userRoles {
			roleNames[j] = role.Name
		}

		// Get work information - get latest (first one in DESC order)
		var workInfoLookup *types.EmployeeWorkInfoLookup
		workInfos, err := s.workInfoRepo.GetByEmployeeID(employee.ID)
		if err == nil && len(workInfos) > 0 {
			// Get the latest work information (first one - already DESC sorted by database)
			latestWorkInfo := workInfos[0]
			workInfoLookup = &types.EmployeeWorkInfoLookup{
				CompanyName:    latestWorkInfo.Company.Name,
				DepartmentName: latestWorkInfo.Department.Name,
				Manager:        latestWorkInfo.Department.Manager,
				JobTitle:       latestWorkInfo.JobPosition.Title,
			}
		}

		// Convert dates to string format for JSON response
		var dateOfBirthStr, hireDateStr, leaveDateStr, professionStartDateStr *string

		if employee.DateOfBirth != nil {
			dateStr := employee.DateOfBirth.Format(time.RFC3339)
			dateOfBirthStr = &dateStr
		}

		if employee.HireDate != nil {
			hireStr := employee.HireDate.Format(time.RFC3339)
			hireDateStr = &hireStr
		}

		if employee.LeaveDate != nil {
			leaveStr := employee.LeaveDate.Format(time.RFC3339)
			leaveDateStr = &leaveStr
		}

		if employee.ProfessionStartDate != nil {
			profStr := employee.ProfessionStartDate.Format(time.RFC3339)
			professionStartDateStr = &profStr
		}

		userInfo := types.UserInfo{
			ID:    user.ID,
			Email: user.Email,
		}

		responses[i] = &types.EmployeeResponse{
			ID:                       employee.ID,
			User:                     userInfo,
			FirstName:                employee.FirstName,
			LastName:                 employee.LastName,
			Email:                    employee.Email,
			CompanyEmail:             employee.CompanyEmail,
			Phone:                    employee.Phone,
			Address:                  employee.Address,
			State:                    employee.State,
			City:                     employee.City,
			Gender:                   employee.Gender,
			DateOfBirth:              dateOfBirthStr,
			HireDate:                 hireDateStr,
			LeaveDate:                leaveDateStr,
			TotalGap:                 employee.TotalGap,
			MaritalStatus:            employee.MaritalStatus,
			EmergencyContact:         employee.EmergencyContact,
			EmergencyContactName:     employee.EmergencyContactName,
			EmergencyContactRelation: employee.EmergencyContactRelation,
			GradeID:                  employee.GradeID,
			ContractNo:               employee.ContractNo,
			ProfessionStartDate:      professionStartDateStr,
			Note:                     employee.Note,
			MotherName:               employee.MotherName,
			FatherName:               employee.FatherName,
			Nationality:              employee.Nationality,
			IdentityNo:               employee.IdentityNo,
			Roles:                    roleNames,
			WorkInformation:          workInfoLookup,
			Status:                   employee.Status, // Include STATUS in the response
		}
	}

	return responses, nil
}

// GetTotalCount returns the total number of employees
func (s *employeeService) GetTotalCount() (int64, error) {
	count, err := s.employeeRepo.GetTotalCount()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTotalCountWithFilters returns the total number of employees with filters applied
func (s *employeeService) GetTotalCountWithFilters(filters map[string]interface{}) (int64, error) {
	count, err := s.employeeRepo.GetTotalCountWithFilters(filters)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetEmployeeCountByGender returns employee count grouped by gender
func (s *employeeService) GetEmployeeCountByGender() ([]interface{}, error) {
	return s.employeeRepo.GetEmployeeCountByGender()
}

// GetEmployeeCountByPosition returns employee count grouped by job position
func (s *employeeService) GetEmployeeCountByPosition() ([]interface{}, error) {
	return s.employeeRepo.GetEmployeeCountByPosition()
}

// GetEmployeeCountByCompanyDepartment returns employee count grouped by company and department
func (s *employeeService) GetEmployeeCountByCompanyDepartment() ([]interface{}, error) {
	return s.employeeRepo.GetEmployeeCountByCompanyDepartment()
}

// GetEmployeeCountByGrade returns employee count grouped by grade
func (s *employeeService) GetEmployeeCountByGrade() ([]interface{}, error) {
	return s.employeeRepo.GetEmployeeCountByGrade()
}

// assignRolesToUser assigns roles to a user based on role names
func (s *employeeService) assignRolesToUser(userID uint, roleNames []string, createdBy string) error {
	if len(roleNames) == 0 {
		return nil
	}

	successCount := 0
	for _, roleName := range roleNames {
		// Get role by name dynamically
		role, err := s.roleRepo.GetByName(roleName)
		if err != nil {
			fmt.Printf("ERROR: role '%s' not found: %v\n", roleName, err)
			return fmt.Errorf("role '%s' not found in database", roleName)
		}

		fmt.Printf("Found role '%s' with ID %d\n", roleName, role.ID)

		userRole := &domain.UserRole{
			UserID: userID,
			RoleID: role.ID,
		}

		if err := s.userRoleRepo.Create(userRole, createdBy); err != nil {
			fmt.Printf("ERROR: failed to create user role for user %d, role %s (roleID: %d): %v\n", userID, roleName, role.ID, err)
			return fmt.Errorf("failed to assign role '%s' to user: %w", roleName, err)
		}

		fmt.Printf("Successfully assigned role '%s' (ID: %d) to user %d\n", roleName, role.ID, userID)
		successCount++
	}

	fmt.Printf("Successfully assigned all %d roles to user %d\n", successCount, userID)
	return nil
}

// updateUserRoles updates roles for a user (deletes old roles and assigns new ones)
func (s *employeeService) updateUserRoles(userID uint, roleNames []string, modifiedBy string) error {
	// First, delete all existing roles for this user
	if err := s.userRoleRepo.DeleteByUserID(userID, modifiedBy); err != nil {
		return fmt.Errorf("failed to delete existing roles: %w", err)
	}

	// Then assign new roles
	return s.assignRolesToUser(userID, roleNames, modifiedBy)
}
