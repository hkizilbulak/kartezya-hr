package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"

	"golang.org/x/crypto/bcrypt"
)

type EmployeeService interface {
	CreateEmployee(companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalExperience float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation, createdBy string) (*domain.Employee, error)
	GetEmployeeByID(id uint) (*types.EmployeeResponse, error)
	GetEmployeeByUserID(userID uint) (*types.EmployeeResponse, error)
	UpdateEmployee(id uint, companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalExperience float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation, modifiedBy string, requestingUserID uint, isAdmin bool) error
	DeleteEmployee(id uint, deletedBy string, isAdmin bool) error
	ListEmployees(limit, offset int, isAdmin bool) ([]*domain.Employee, error)
}

type employeeService struct {
	employeeRepo repository.EmployeeRepository
	userRepo     repository.UserRepository
	auditService AuditService
}

func NewEmployeeService(employeeRepo repository.EmployeeRepository, userRepo repository.UserRepository, auditService AuditService) EmployeeService {
	return &employeeService{
		employeeRepo: employeeRepo,
		userRepo:     userRepo,
		auditService: auditService,
	}
}

func (s *employeeService) CreateEmployee(companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalExperience float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation, createdBy string) (*domain.Employee, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(companyEmail)
	var user *domain.User

	if err != nil {
		// User doesn't exist, create new user with default password "employee123"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("employee123"), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %v", err)
		}

		user = &domain.User{
			Email:    companyEmail,
			Password: string(hashedPassword),
		}

		err = s.userRepo.Create(user, createdBy)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %v", err)
		}
	} else {
		user = existingUser
	}

	// Parse date fields
	var dateOfBirthPtr, hireDatePtr, leaveDatePtr *time.Time

	if dateOfBirth != "" {
		if parsed, err := time.Parse("2006-01-02", dateOfBirth); err == nil {
			dateOfBirthPtr = &parsed
		}
	}

	if hireDate != "" {
		if parsed, err := time.Parse("2006-01-02", hireDate); err == nil {
			hireDatePtr = &parsed
		}
	}

	if leaveDate != "" {
		if parsed, err := time.Parse("2006-01-02", leaveDate); err == nil {
			leaveDatePtr = &parsed
		}
	}

	// Create employee profile
	employee := &domain.Employee{
		UserID:                   user.ID,
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
		TotalExperience:          totalExperience,
		MaritalStatus:            maritalStatus,
		EmergencyContact:         emergencyContact,
		EmergencyContactName:     emergencyContactName,
		EmergencyContactRelation: emergencyContactRelation,
	}

	// Create the employee
	if err := s.employeeRepo.Create(employee, createdBy); err != nil {
		return nil, fmt.Errorf("failed to create employee: %v", err)
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("Employee", employee.ID, domain.AuditActionCreate, nil, employee, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return employee, nil
}

func (s *employeeService) GetEmployeeByID(id uint) (*types.EmployeeResponse, error) {
	employee, err := s.employeeRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(employee.UserID)
	if err != nil {
		return nil, err
	}

	// Convert dates to string format for JSON response
	var dateOfBirthStr, hireDateStr, leaveDateStr *string

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

	userInfo := types.UserInfo{
		ID:    user.ID,
		Email: user.Email,
	}

	return &types.EmployeeResponse{
		ID:                       employee.ID,
		User:                     userInfo,
		FirstName:                employee.FirstName,
		LastName:                 employee.LastName,
		Phone:                    employee.Phone,
		Address:                  employee.Address,
		State:                    employee.State,
		City:                     employee.City,
		Gender:                   employee.Gender,
		DateOfBirth:              dateOfBirthStr,
		HireDate:                 hireDateStr,
		LeaveDate:                leaveDateStr,
		TotalExperience:          employee.TotalExperience,
		MaritalStatus:            employee.MaritalStatus,
		EmergencyContact:         employee.EmergencyContact,
		EmergencyContactName:     employee.EmergencyContactName,
		EmergencyContactRelation: employee.EmergencyContactRelation,
	}, nil
}

func (s *employeeService) GetEmployeeByUserID(userID uint) (*types.EmployeeResponse, error) {
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(employee.UserID)
	if err != nil {
		return nil, err
	}

	// Convert dates to string format for JSON response
	var dateOfBirthStr, hireDateStr, leaveDateStr *string

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

	userInfo := types.UserInfo{
		ID:    user.ID,
		Email: user.Email,
	}

	return &types.EmployeeResponse{
		ID:                       employee.ID,
		User:                     userInfo,
		FirstName:                employee.FirstName,
		LastName:                 employee.LastName,
		Phone:                    employee.Phone,
		Address:                  employee.Address,
		State:                    employee.State,
		City:                     employee.City,
		Gender:                   employee.Gender,
		DateOfBirth:              dateOfBirthStr,
		HireDate:                 hireDateStr,
		LeaveDate:                leaveDateStr,
		TotalExperience:          employee.TotalExperience,
		MaritalStatus:            employee.MaritalStatus,
		EmergencyContact:         employee.EmergencyContact,
		EmergencyContactName:     employee.EmergencyContactName,
		EmergencyContactRelation: employee.EmergencyContactRelation,
	}, nil
}

func (s *employeeService) UpdateEmployee(id uint, companyEmail, firstName, lastName, phone, address, state, city, gender, dateOfBirth, hireDate, leaveDate string, totalExperience float64, maritalStatus, emergencyContact, emergencyContactName, emergencyContactRelation, modifiedBy string, requestingUserID uint, isAdmin bool) error {
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

	// Parse date fields
	var dateOfBirthPtr, hireDatePtr, leaveDatePtr *time.Time

	if dateOfBirth != "" {
		if parsed, err := time.Parse("2006-01-02", dateOfBirth); err == nil {
			dateOfBirthPtr = &parsed
		}
	}

	if hireDate != "" {
		if parsed, err := time.Parse("2006-01-02", hireDate); err == nil {
			hireDatePtr = &parsed
		}
	}

	if leaveDate != "" {
		if parsed, err := time.Parse("2006-01-02", leaveDate); err == nil {
			leaveDatePtr = &parsed
		}
	}

	// Create updated employee object
	employee := &domain.Employee{
		UserID:                   existingEmployee.UserID, // Preserve existing user ID
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
		TotalExperience:          totalExperience,
		MaritalStatus:            maritalStatus,
		EmergencyContact:         emergencyContact,
		EmergencyContactName:     emergencyContactName,
		EmergencyContactRelation: emergencyContactRelation,
	}

	// Set the ID after creating the struct
	employee.ID = id

	// Update employee
	if err := s.employeeRepo.Update(employee, modifiedBy); err != nil {
		return err
	}

	// Get updated employee for audit
	updatedEmployee, _ := s.employeeRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("Employee", id, domain.AuditActionUpdate, existingEmployee, updatedEmployee, modifiedBy); err != nil {
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

func (s *employeeService) ListEmployees(limit, offset int, isAdmin bool) ([]*domain.Employee, error) {
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

	return employees, nil
}
