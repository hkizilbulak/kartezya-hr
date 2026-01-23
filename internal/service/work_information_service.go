package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type WorkInformationService interface {
	CreateWorkInformation(employeeID, companyID, departmentID, jobPositionID uint, startDate, endDate, personnelNo, workEmail, createdBy string) (*domain.EmployeeWorkInformation, error)
	GetWorkInformationByID(id uint) (*types.WorkInformationResponse, error)
	GetWorkInformationByUserID(userID uint) ([]*types.WorkInformationWithNames, error)
	UpdateWorkInformation(id uint, employeeID, companyID, departmentID, jobPositionID uint, startDate, endDate, personnelNo, workEmail, modifiedBy string, requestingUserID uint, isAdmin bool) error
	DeleteWorkInformation(id uint, deletedBy string) error
	GetAllWorkInformations(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error)
}

type workInformationService struct {
	workInfoRepo    repository.WorkInformationRepository
	employeeRepo    repository.EmployeeRepository
	companyRepo     repository.CompanyRepository
	departmentRepo  repository.DepartmentRepository
	jobPositionRepo repository.JobPositionRepository
	auditService    AuditService
}

func NewWorkInformationService(workInfoRepo repository.WorkInformationRepository, employeeRepo repository.EmployeeRepository, companyRepo repository.CompanyRepository, departmentRepo repository.DepartmentRepository, jobPositionRepo repository.JobPositionRepository, auditService AuditService) WorkInformationService {
	return &workInformationService{
		workInfoRepo:    workInfoRepo,
		employeeRepo:    employeeRepo,
		companyRepo:     companyRepo,
		departmentRepo:  departmentRepo,
		jobPositionRepo: jobPositionRepo,
		auditService:    auditService,
	}
}

func (s *workInformationService) CreateWorkInformation(employeeID, companyID, departmentID, jobPositionID uint, startDate, endDate, personnelNo, workEmail, createdBy string) (*domain.EmployeeWorkInformation, error) {
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

	// Create work information
	workInfo := &domain.EmployeeWorkInformation{
		EmployeeID:    employeeID,
		CompanyID:     companyID,
		DepartmentID:  departmentID,
		JobPositionID: jobPositionID,
		StartDate:     startDatePtr,
		EndDate:       endDatePtr,
		PersonnelNo:   personnelNo,
		WorkEmail:     workEmail,
	}

	// Create the work information
	if err := s.workInfoRepo.Create(workInfo, createdBy); err != nil {
		return nil, fmt.Errorf("failed to create work information: %v", err)
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("EmployeeWorkInformation", workInfo.ID, domain.AuditActionCreate, nil, workInfo, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return workInfo, nil
}

func (s *workInformationService) GetWorkInformationByID(id uint) (*types.WorkInformationResponse, error) {

	if id == 0 {
		return nil, errors.New("invalid work information ID")
	}

	workInfo, err := s.workInfoRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return &types.WorkInformationResponse{
		ID:          workInfo.ID,
		CreatedAt:   workInfo.CreatedAt,
		UpdatedAt:   workInfo.UpdatedAt,
		Deleted:     workInfo.Deleted,
		CreatedBy:   workInfo.CreatedBy,
		ModifiedBy:  workInfo.ModifiedBy,
		StartDate:   workInfo.StartDate,
		EndDate:     workInfo.EndDate,
		PersonnelNo: workInfo.PersonnelNo,
		WorkEmail:   workInfo.WorkEmail,
		Employee: types.WorkInformationEmployeeLookup{
			ID:        workInfo.Employee.ID,
			FirstName: workInfo.Employee.FirstName,
			LastName:  workInfo.Employee.LastName,
		},
		Company: types.WorkInformationCompanyLookup{
			ID:   workInfo.Company.ID,
			Name: workInfo.Company.Name,
		},
		Department: types.WorkInformationDepartmentLookup{
			ID:      workInfo.Department.ID,
			Name:    workInfo.Department.Name,
			Manager: workInfo.Department.Manager,
		},
		JobPosition: types.WorkInformationJobPositionLookup{
			ID:    workInfo.JobPosition.ID,
			Title: workInfo.JobPosition.Title,
		},
	}, nil
}

func (s *workInformationService) GetWorkInformationByUserID(userID uint) ([]*types.WorkInformationWithNames, error) {
	// Get work information records for the user
	workInfos, err := s.workInfoRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	var result []*types.WorkInformationWithNames

	for _, workInfo := range workInfos {
		// Get company name
		company, err := s.companyRepo.GetByID(workInfo.CompanyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get company: %v", err)
		}

		// Get department name and manager
		department, err := s.departmentRepo.GetByID(workInfo.DepartmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get department: %v", err)
		}

		// Get job position name
		jobPosition, err := s.jobPositionRepo.GetByID(workInfo.JobPositionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get job position: %v", err)
		}

		var startDateStr string
		var endDateStr *string

		// Check if StartDate is not zero value instead of comparing with nil
		if !workInfo.StartDate.IsZero() {
			startDateStr = workInfo.StartDate.Format(time.RFC3339)
		}

		if workInfo.EndDate != nil {
			dateStr := workInfo.EndDate.Format(time.RFC3339)
			endDateStr = &dateStr
		}

		// Create DTO with related entity names
		workInfoDTO := &types.WorkInformationWithNames{
			ID:              workInfo.ID,
			CompanyName:     company.Name,
			DepartmentName:  department.Name,
			Manager:         department.Manager,
			JobPositionName: jobPosition.Title,
			StartDate:       startDateStr,
			EndDate:         endDateStr,
		}

		result = append(result, workInfoDTO)
	}

	return result, nil
}

func (s *workInformationService) UpdateWorkInformation(id uint, employeeID, companyID, departmentID, jobPositionID uint, startDate, endDate, personnelNo, workEmail, modifiedBy string, requestingUserID uint, isAdmin bool) error {
	// Get existing work information for audit trail
	existingWorkInfo, err := s.workInfoRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Authorization check: Non-admin users can only update their own work information
	if !isAdmin {
		// Get the employee record associated with the work information to get the UserID
		employee, err := s.employeeRepo.GetByID(existingWorkInfo.EmployeeID)
		if err != nil {
			return fmt.Errorf("failed to get employee for authorization: %v", err)
		}

		if employee.UserID != requestingUserID {
			return errors.New("unauthorized to update this work information")
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

	// Create updated work information object
	workInfo := &domain.EmployeeWorkInformation{
		EmployeeID:    employeeID,
		CompanyID:     companyID,
		DepartmentID:  departmentID,
		JobPositionID: jobPositionID,
		StartDate:     startDatePtr,
		EndDate:       endDatePtr,
		PersonnelNo:   personnelNo,
		WorkEmail:     workEmail,
	}

	// Set the ID after creating the struct
	workInfo.ID = id

	// Update work information
	if err := s.workInfoRepo.Update(workInfo, modifiedBy); err != nil {
		return err
	}

	// Get updated work information for audit
	updatedWorkInfo, _ := s.workInfoRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("EmployeeWorkInformation", id, domain.AuditActionUpdate, existingWorkInfo, updatedWorkInfo, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *workInformationService) DeleteWorkInformation(id uint, deletedBy string) error {
	// Get existing work information for audit trail
	existingWorkInfo, err := s.workInfoRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the work information
	if err := s.workInfoRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("EmployeeWorkInformation", id, domain.AuditActionDelete, existingWorkInfo, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *workInformationService) GetAllWorkInformations(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error) {

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
	workInfos, total, err := s.workInfoRepo.GetAll(limit, offset, sortParams, employeeID)
	if err != nil {
		return nil, err
	}

	workInfoResponses := make([]types.WorkInformationResponse, len(workInfos))
	for i, workInfo := range workInfos {
		workInfoResponses[i] = types.WorkInformationResponse{
			ID:          workInfo.ID,
			CreatedAt:   workInfo.CreatedAt,
			UpdatedAt:   workInfo.UpdatedAt,
			Deleted:     workInfo.Deleted,
			CreatedBy:   workInfo.CreatedBy,
			ModifiedBy:  workInfo.ModifiedBy,
			StartDate:   workInfo.StartDate,
			EndDate:     workInfo.EndDate,
			PersonnelNo: workInfo.PersonnelNo,
			WorkEmail:   workInfo.WorkEmail,
			Employee: types.WorkInformationEmployeeLookup{
				ID:        workInfo.Employee.ID,
				FirstName: workInfo.Employee.FirstName,
				LastName:  workInfo.Employee.LastName,
			},
			Company: types.WorkInformationCompanyLookup{
				ID:   workInfo.Company.ID,
				Name: workInfo.Company.Name,
			},
			Department: types.WorkInformationDepartmentLookup{
				ID:      workInfo.Department.ID,
				Name:    workInfo.Department.Name,
				Manager: workInfo.Department.Manager,
			},
			JobPosition: types.WorkInformationJobPositionLookup{
				ID:    workInfo.JobPosition.ID,
				Title: workInfo.JobPosition.Title,
			},
		}
	}

	return &PaginatedResponse{
		Data: workInfoResponses,
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
