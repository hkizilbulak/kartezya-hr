package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type EmployeeGradeService interface {
	CreateEmployeeGrade(employeeID, gradeID uint, startDate, endDate, createdBy string) (*domain.EmployeeGrade, error)
	GetEmployeeGradeByID(id uint) (*types.EmployeeGradeResponse, error)
	GetEmployeeGradeByUserID(userID uint) ([]*types.EmployeeGradeWithNames, error)
	UpdateEmployeeGrade(id uint, employeeID, gradeID uint, startDate, endDate, modifiedBy string, requestingUserID uint, isAdmin bool) error
	DeleteEmployeeGrade(id uint, deletedBy string) error
	GetAllEmployeeGrades(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error)
}

type employeeGradeService struct {
	employeeGradeRepo repository.EmployeeGradeRepository
	employeeRepo      repository.EmployeeRepository
	gradeRepo         repository.GradeRepository
	auditService      AuditService
}

func NewEmployeeGradeService(employeeGradeRepo repository.EmployeeGradeRepository, employeeRepo repository.EmployeeRepository, gradeRepo repository.GradeRepository, auditService AuditService) EmployeeGradeService {
	return &employeeGradeService{
		employeeGradeRepo: employeeGradeRepo,
		employeeRepo:      employeeRepo,
		gradeRepo:         gradeRepo,
		auditService:      auditService,
	}
}

func (s *employeeGradeService) CreateEmployeeGrade(employeeID, gradeID uint, startDate, endDate, createdBy string) (*domain.EmployeeGrade, error) {
	// Parse date fields
	var startDatePtr time.Time
	var endDatePtr *time.Time

	if parsed, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("invalid start date format: %v", err)
	} else {
		startDatePtr = parsed
	}

	if endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err != nil {
			return nil, fmt.Errorf("invalid end date format: %v", err)
		} else {
			endDatePtr = &parsed
		}
	}

	// Create employee grade
	employeeGrade := &domain.EmployeeGrade{
		EmployeeID: employeeID,
		GradeID:    gradeID,
		StartDate:  startDatePtr,
		EndDate:    endDatePtr,
	}

	// Create the employee grade
	if err := s.employeeGradeRepo.Create(employeeGrade, createdBy); err != nil {
		return nil, fmt.Errorf("failed to create employee grade: %v", err)
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("EmployeeGrade", employeeGrade.ID, domain.AuditActionCreate, nil, employeeGrade, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return employeeGrade, nil
}

func (s *employeeGradeService) GetEmployeeGradeByID(id uint) (*types.EmployeeGradeResponse, error) {
	if id == 0 {
		return nil, errors.New("invalid employee grade ID")
	}

	employeeGrade, err := s.employeeGradeRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return &types.EmployeeGradeResponse{
		ID:         employeeGrade.ID,
		CreatedAt:  employeeGrade.CreatedAt,
		UpdatedAt:  employeeGrade.UpdatedAt,
		Deleted:    employeeGrade.Deleted,
		CreatedBy:  employeeGrade.CreatedBy,
		ModifiedBy: employeeGrade.ModifiedBy,
		StartDate:  employeeGrade.StartDate,
		EndDate:    employeeGrade.EndDate,
		Employee: types.EmployeeLookup{
			ID:        employeeGrade.Employee.ID,
			FirstName: employeeGrade.Employee.FirstName,
			LastName:  employeeGrade.Employee.LastName,
		},
		Grade: types.GradeLookup{
			ID:   employeeGrade.Grade.ID,
			Name: employeeGrade.Grade.Name,
		},
	}, nil
}

func (s *employeeGradeService) GetEmployeeGradeByUserID(userID uint) ([]*types.EmployeeGradeWithNames, error) {
	// Get employee grade records for the user
	employeeGrades, err := s.employeeGradeRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	var result []*types.EmployeeGradeWithNames

	for _, employeeGrade := range employeeGrades {
		// Get grade name
		grade, err := s.gradeRepo.GetByID(int64(employeeGrade.GradeID))
		if err != nil {
			return nil, fmt.Errorf("failed to get grade: %v", err)
		}

		var startDateStr string
		var endDateStr *string

		// Check if StartDate is not zero value
		if !employeeGrade.StartDate.IsZero() {
			startDateStr = employeeGrade.StartDate.Format(time.RFC3339)
		}

		if employeeGrade.EndDate != nil {
			dateStr := employeeGrade.EndDate.Format(time.RFC3339)
			endDateStr = &dateStr
		}

		// Create DTO with related entity names
		employeeGradeDTO := &types.EmployeeGradeWithNames{
			ID:        employeeGrade.ID,
			GradeName: grade.Name,
			StartDate: startDateStr,
			EndDate:   endDateStr,
		}

		result = append(result, employeeGradeDTO)
	}

	return result, nil
}

func (s *employeeGradeService) UpdateEmployeeGrade(id uint, employeeID, gradeID uint, startDate, endDate, modifiedBy string, requestingUserID uint, isAdmin bool) error {
	// Get existing employee grade for audit trail
	existingEmployeeGrade, err := s.employeeGradeRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Authorization check: Non-admin users can only update their own employee grades
	if !isAdmin {
		// Get the employee record associated with the grade to get the UserID
		employee, err := s.employeeRepo.GetByID(existingEmployeeGrade.EmployeeID)
		if err != nil {
			return fmt.Errorf("failed to get employee for authorization: %v", err)
		}

		if employee.UserID != requestingUserID {
			return errors.New("unauthorized to update this employee grade")
		}
	}

	// Parse date fields
	var startDatePtr time.Time
	var endDatePtr *time.Time

	if parsed, err := time.Parse("2006-01-02", startDate); err != nil {
		return fmt.Errorf("invalid start date format: %v", err)
	} else {
		startDatePtr = parsed
	}

	if endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err != nil {
			return fmt.Errorf("invalid end date format: %v", err)
		} else {
			endDatePtr = &parsed
		}
	}

	// Create updated employee grade object
	employeeGrade := &domain.EmployeeGrade{
		EmployeeID: employeeID,
		GradeID:    gradeID,
		StartDate:  startDatePtr,
		EndDate:    endDatePtr,
	}

	// Set the ID after creating the struct
	employeeGrade.ID = id

	// Update employee grade
	if err := s.employeeGradeRepo.Update(employeeGrade, modifiedBy); err != nil {
		return err
	}

	// Get updated employee grade for audit
	updatedEmployeeGrade, _ := s.employeeGradeRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("EmployeeGrade", id, domain.AuditActionUpdate, existingEmployeeGrade, updatedEmployeeGrade, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *employeeGradeService) DeleteEmployeeGrade(id uint, deletedBy string) error {
	// Get existing employee grade for audit trail
	existingEmployeeGrade, err := s.employeeGradeRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the employee grade
	if err := s.employeeGradeRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("EmployeeGrade", id, domain.AuditActionDelete, existingEmployeeGrade, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *employeeGradeService) GetAllEmployeeGrades(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Set defaults for sorting
	if sortParams.Sort == "" {
		sortParams.Sort = "id"
	}
	if sortParams.Direction == "" {
		sortParams.Direction = "ASC"
	}

	offset := (page - 1) * limit
	employeeGrades, total, err := s.employeeGradeRepo.GetAll(limit, offset, sortParams, employeeID)
	if err != nil {
		return nil, err
	}

	employeeGradeResponses := make([]types.EmployeeGradeResponse, len(employeeGrades))
	for i, employeeGrade := range employeeGrades {
		employeeGradeResponses[i] = types.EmployeeGradeResponse{
			ID:         employeeGrade.ID,
			CreatedAt:  employeeGrade.CreatedAt,
			UpdatedAt:  employeeGrade.UpdatedAt,
			Deleted:    employeeGrade.Deleted,
			CreatedBy:  employeeGrade.CreatedBy,
			ModifiedBy: employeeGrade.ModifiedBy,
			StartDate:  employeeGrade.StartDate,
			EndDate:    employeeGrade.EndDate,
			Employee: types.EmployeeLookup{
				ID:        employeeGrade.Employee.ID,
				FirstName: employeeGrade.Employee.FirstName,
				LastName:  employeeGrade.Employee.LastName,
			},
			Grade: types.GradeLookup{
				ID:   employeeGrade.Grade.ID,
				Name: employeeGrade.Grade.Name,
			},
		}
	}

	return &PaginatedResponse{
		Data: employeeGradeResponses,
		Page: PageInfo{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: (total + int64(limit) - 1) / int64(limit),
			Sort:       sortParams.Sort,
			Direction:  sortParams.Direction,
		},
	}, nil
}
