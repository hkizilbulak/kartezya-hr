package service

import (
	"errors"
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type DepartmentService interface {
	CreateDepartment(department *domain.Department, userID uint) error
	GetDepartmentByID(id uint) (*types.DepartmentResponse, error)
	GetAllDepartments(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	UpdateDepartment(id uint, department *domain.Department, userID uint) error
	DeleteDepartment(id uint, userID uint) error
	GetTotalCount() (int64, error)
}

type departmentService struct {
	departmentRepo repository.DepartmentRepository
	companyRepo    repository.CompanyRepository
	auditService   AuditService
}

func NewDepartmentService(departmentRepo repository.DepartmentRepository, companyRepo repository.CompanyRepository, auditService AuditService) DepartmentService {
	return &departmentService{
		departmentRepo: departmentRepo,
		companyRepo:    companyRepo,
		auditService:   auditService,
	}
}

func (s *departmentService) CreateDepartment(department *domain.Department, userID uint) error {
	// Validation
	if department.Name == "" {
		return errors.New("department name is required")
	}
	if department.CompanyID == 0 {
		return errors.New("company ID is required")
	}

	// Check if company exists
	_, err := s.companyRepo.GetByID(department.CompanyID)
	if err != nil {
		return errors.New("company not found")
	}

	// Check if a department with the same name already exists in this company
	existingDepartment, err := s.departmentRepo.GetByCompanyIDAndName(department.CompanyID, department.Name)
	if err == nil && existingDepartment != nil {
		return errors.New("department with this name already exists in the company")
	}

	// Create audit identifier
	createdBy := fmt.Sprintf("%d", userID)

	// Create the department
	if err := s.departmentRepo.Create(department, createdBy); err != nil {
		return err
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("Department", department.ID, domain.AuditActionCreate, nil, department, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *departmentService) GetDepartmentByID(id uint) (*types.DepartmentResponse, error) {
	if id == 0 {
		return nil, errors.New("invalid department ID")
	}

	department, err := s.departmentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Get company info for the response
	company, err := s.companyRepo.GetByID(department.CompanyID)
	if err != nil {
		return nil, errors.New("company not found")
	}

	// Transform to response DTO with nested company object
	return &types.DepartmentResponse{
		ID:         department.ID,
		CreatedAt:  department.CreatedAt,
		UpdatedAt:  department.UpdatedAt,
		Deleted:    department.Deleted,
		CreatedBy:  department.CreatedBy,
		ModifiedBy: department.ModifiedBy,
		Name:       department.Name,
		Manager:    department.Manager,
		Company: types.CompanyLookup{
			ID:   company.ID,
			Name: company.Name,
		},
	}, nil
}

func (s *departmentService) GetAllDepartments(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
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
	departments, total, err := s.departmentRepo.GetAll(limit, offset, sortParams)
	if err != nil {
		return nil, err
	}

	// Transform domain models to response DTOs with nested company objects
	departmentResponses := make([]types.DepartmentResponse, len(departments))
	for i, department := range departments {
		// Get company info for each department
		company, err := s.companyRepo.GetByID(department.CompanyID)
		if err != nil {
			return nil, err
		}

		departmentResponses[i] = types.DepartmentResponse{
			ID:         department.ID,
			CreatedAt:  department.CreatedAt,
			UpdatedAt:  department.UpdatedAt,
			Deleted:    department.Deleted,
			CreatedBy:  department.CreatedBy,
			ModifiedBy: department.ModifiedBy,
			Name:       department.Name,
			Manager:    department.Manager,
			Company: types.CompanyLookup{
				ID:   company.ID,
				Name: company.Name,
			},
		}
	}

	return &PaginatedResponse{
		Data: departmentResponses,
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

func (s *departmentService) UpdateDepartment(id uint, department *domain.Department, userID uint) error {
	if id == 0 {
		return errors.New("invalid department ID")
	}

	// Check if department exists and get old value for audit
	existingDepartment, err := s.departmentRepo.GetByID(id)
	if err != nil {
		return errors.New("department not found")
	}
	if existingDepartment == nil {
		return errors.New("department not found")
	}

	// Validation
	if department.Name == "" {
		return errors.New("department name is required")
	}
	if department.CompanyID == 0 {
		return errors.New("company ID is required")
	}

	// Check if company exists
	_, err = s.companyRepo.GetByID(department.CompanyID)
	if err != nil {
		return errors.New("company not found")
	}

	// Check if the new name or company is different from the current values
	if existingDepartment.Name != department.Name || existingDepartment.CompanyID != department.CompanyID {
		// Check if another department with the same name already exists in the target company
		existingWithName, err := s.departmentRepo.GetByCompanyIDAndName(department.CompanyID, department.Name)
		if err == nil && existingWithName != nil && existingWithName.ID != id {
			return errors.New("department with this name already exists in the company")
		}
	}

	// Create audit identifier
	modifiedBy := fmt.Sprintf("%d", userID)

	// Clone the existing department and update only the provided fields
	updatedDepartment := *existingDepartment
	updatedDepartment.Name = department.Name
	updatedDepartment.Manager = department.Manager
	updatedDepartment.CompanyID = department.CompanyID

	// Update the department
	if err := s.departmentRepo.Update(&updatedDepartment, modifiedBy); err != nil {
		return err
	}

	// Get updated department for audit
	auditedDepartment, _ := s.departmentRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("Department", id, domain.AuditActionUpdate, existingDepartment, auditedDepartment, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *departmentService) DeleteDepartment(id uint, userID uint) error {
	if id == 0 {
		return errors.New("invalid department ID")
	}

	// Check if department exists and get old value for audit
	existingDepartment, err := s.departmentRepo.GetByID(id)
	if err != nil {
		return errors.New("department not found")
	}
	if existingDepartment == nil {
		return errors.New("department not found")
	}

	// Check if department has employees (through work information)
	if len(existingDepartment.EmployeeWorkInformation) > 0 {
		return errors.New("cannot delete department with existing employees")
	}

	// Create audit identifier
	deletedBy := fmt.Sprintf("%d", userID)

	// Delete the department
	if err := s.departmentRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("Department", id, domain.AuditActionDelete, existingDepartment, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

// GetTotalCount returns the total number of departments
func (s *departmentService) GetTotalCount() (int64, error) {
	count, err := s.departmentRepo.GetTotalCount()
	if err != nil {
		return 0, err
	}
	return count, nil
}
