package service

import (
	"errors"
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type CompanyService interface {
	CreateCompany(company *domain.Company, userID uint) error
	GetCompanyByID(id uint) (*types.CompanyResponse, error)
	GetAllCompanies(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	UpdateCompany(id uint, company *domain.Company, userID uint) error
	DeleteCompany(id uint, userID uint) error
	GetDepartmentsByCompanyLookup(companyID uint) ([]types.DepartmentLookup, error)
	GetTotalCount() (int64, error)
}

type companyService struct {
	companyRepo       repository.CompanyRepository
	departmentRepo    repository.DepartmentRepository
	departmentService DepartmentService
	auditService      AuditService
}

func NewCompanyService(companyRepo repository.CompanyRepository, departmentRepo repository.DepartmentRepository, departmentService DepartmentService, auditService AuditService) CompanyService {
	return &companyService{
		companyRepo:       companyRepo,
		departmentRepo:    departmentRepo,
		departmentService: departmentService,
		auditService:      auditService,
	}
}

func (s *companyService) CreateCompany(company *domain.Company, userID uint) error {
	// Validation
	if company.Name == "" {
		return errors.New("company name is required")
	}

	// Check if a company with the same name already exists
	existingCompany, err := s.companyRepo.GetByName(company.Name)
	if err == nil && existingCompany != nil {
		return errors.New("company with this name already exists")
	}

	// Create audit identifier
	createdBy := fmt.Sprintf("%d", userID)

	// Create the company
	if err := s.companyRepo.Create(company, createdBy); err != nil {
		return err
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("Company", company.ID, domain.AuditActionCreate, nil, company, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *companyService) GetCompanyByID(id uint) (*types.CompanyResponse, error) {
	if id == 0 {
		return nil, errors.New("invalid company ID")
	}

	company, err := s.companyRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return &types.CompanyResponse{
		ID:         company.ID,
		CreatedAt:  company.CreatedAt,
		UpdatedAt:  company.UpdatedAt,
		Deleted:    company.Deleted,
		CreatedBy:  company.CreatedBy,
		ModifiedBy: company.ModifiedBy,
		Name:       company.Name,
		Address:    company.Address,
		Phone:      company.Phone,
		Email:      company.Email,
		Website:    company.Website,
	}, nil
}

func (s *companyService) GetAllCompanies(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
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
	companies, total, err := s.companyRepo.GetAll(limit, offset, sortParams)
	if err != nil {
		return nil, err
	}

	// Transform domain models to response DTOs (excluding departments)
	companyResponses := make([]types.CompanyResponse, len(companies))
	for i, company := range companies {
		companyResponses[i] = types.CompanyResponse{
			ID:         company.ID,
			CreatedAt:  company.CreatedAt,
			UpdatedAt:  company.UpdatedAt,
			Deleted:    company.Deleted,
			CreatedBy:  company.CreatedBy,
			ModifiedBy: company.ModifiedBy,
			Name:       company.Name,
			Address:    company.Address,
			Phone:      company.Phone,
			Email:      company.Email,
			Website:    company.Website,
		}
	}

	return &PaginatedResponse{
		Data: companyResponses,
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

func (s *companyService) UpdateCompany(id uint, company *domain.Company, userID uint) error {
	if id == 0 {
		return errors.New("invalid company ID")
	}

	// Check if company exists and get old value for audit
	existingCompany, err := s.companyRepo.GetByID(id)
	if err != nil {
		return errors.New("company not found")
	}
	if existingCompany == nil {
		return errors.New("company not found")
	}

	// Validation
	if company.Name == "" {
		return errors.New("company name is required")
	}

	// Check if the new name is different from the current name
	if existingCompany.Name != company.Name {
		// Check if another company with the same name already exists
		existingWithName, err := s.companyRepo.GetByName(company.Name)
		if err == nil && existingWithName != nil && existingWithName.ID != id {
			return errors.New("company with this name already exists")
		}
	}

	// Create audit identifier
	modifiedBy := fmt.Sprintf("%d", userID)

	// Clone the existing company and update only the provided fields
	updatedCompany := *existingCompany
	updatedCompany.Name = company.Name
	updatedCompany.Address = company.Address
	updatedCompany.Phone = company.Phone
	updatedCompany.Email = company.Email
	updatedCompany.Website = company.Website

	// Update the company
	if err := s.companyRepo.Update(&updatedCompany, modifiedBy); err != nil {
		return err
	}

	// Get updated company for audit
	auditedCompany, _ := s.companyRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("Company", id, domain.AuditActionUpdate, existingCompany, auditedCompany, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *companyService) DeleteCompany(id uint, userID uint) error {
	if id == 0 {
		return errors.New("invalid company ID")
	}

	// Check if company exists and get old value for audit
	existingCompany, err := s.companyRepo.GetByID(id)
	if err != nil {
		return errors.New("company not found")
	}
	if existingCompany == nil {
		return errors.New("company not found")
	}

	// Check if company has departments
	departments, err := s.departmentRepo.GetByCompanyID(id)
	if err != nil {
		return err
	}
	if len(departments) > 0 {
		return errors.New("cannot delete company with existing departments")
	}

	// Create audit identifier
	deletedBy := fmt.Sprintf("%d", userID)

	// Delete the company
	if err := s.companyRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("Company", id, domain.AuditActionDelete, existingCompany, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *companyService) GetDepartmentsByCompanyLookup(companyID uint) ([]types.DepartmentLookup, error) {
	if companyID == 0 {
		return nil, errors.New("invalid company ID")
	}

	// Check if company exists
	_, err := s.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, errors.New("company not found")
	}

	departments, err := s.departmentRepo.GetByCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	// Convert to lookup DTOs
	lookupData := make([]types.DepartmentLookup, len(departments))
	for i, department := range departments {
		lookupData[i] = types.DepartmentLookup{
			ID:      department.ID,
			Name:    department.Name,
			Manager: department.Manager,
		}
	}

	return lookupData, nil
}

// GetTotalCount returns the total number of companies
func (s *companyService) GetTotalCount() (int64, error) {
	count, err := s.companyRepo.GetTotalCount()
	if err != nil {
		return 0, err
	}
	return count, nil
}
