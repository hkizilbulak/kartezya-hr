package service

import (
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type ContractService interface {
	CreateContract(req types.ContractRequest, createdBy string) (*domain.Contract, error)
	GetContractByID(id uint) (*domain.Contract, error)
	GetAllContracts(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	UpdateContract(id uint, req types.ContractRequest, modifiedBy string) error
	DeleteContract(id uint, deletedBy string) error
}

type contractService struct {
	contractRepo repository.ContractRepository
	auditService AuditService
}

func NewContractService(contractRepo repository.ContractRepository, auditService AuditService) ContractService {
	return &contractService{
		contractRepo: contractRepo,
		auditService: auditService,
	}
}

func (s *contractService) CreateContract(req types.ContractRequest, createdBy string) (*domain.Contract, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format")
	}

	var endDatePtr *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end date format")
		}
		endDatePtr = &endDate
	}

	status := req.Status
	if status == "" {
		status = domain.ContractStatusPendingProposal
	}

	contract := &domain.Contract{
		CustomerContactName:  req.CustomerContactName,
		CustomerContactPhone: req.CustomerContactPhone,
		CustomerContactEmail: req.CustomerContactEmail,
		ProjectName:          req.ProjectName,
		ContractNo:           req.ContractNo,
		StartDate:            startDate,
		EndDate:              endDatePtr,
		Status:               status,
	}

	if err := s.contractRepo.Create(contract, createdBy); err != nil {
		return nil, err
	}

	s.auditService.CreateAuditLog("Contract", contract.ID, domain.AuditActionCreate, nil, contract, createdBy)

	return contract, nil
}

func (s *contractService) GetContractByID(id uint) (*domain.Contract, error) {
	return s.contractRepo.GetByID(id)
}

func (s *contractService) GetAllContracts(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	contracts, total, err := s.contractRepo.GetAll(limit, offset, sortParams)
	if err != nil {
		return nil, err
	}

	responses := make([]types.ContractResponse, len(contracts))
	for i, c := range contracts {
		var endDateStr *string
		if c.EndDate != nil {
			str := c.EndDate.Format("2006-01-02")
			endDateStr = &str
		}

		responses[i] = types.ContractResponse{
			ID:                   c.ID,
			CreatedAt:            c.CreatedAt.Format(time.RFC3339),
			UpdatedAt:            c.UpdatedAt.Format(time.RFC3339),
			CustomerContactName:  c.CustomerContactName,
			CustomerContactPhone: c.CustomerContactPhone,
			CustomerContactEmail: c.CustomerContactEmail,
			ProjectName:          c.ProjectName,
			ContractNo:           c.ContractNo,
			StartDate:            c.StartDate.Format("2006-01-02"),
			EndDate:              endDateStr,
			Status:               c.Status,
		}
	}

	return &PaginatedResponse{
		Data: responses,
		Page: PageInfo{Total: total, Page: page, Limit: limit},
	}, nil
}

func (s *contractService) UpdateContract(id uint, req types.ContractRequest, modifiedBy string) error {
	existing, err := s.contractRepo.GetByID(id)
	if err != nil {
		return err
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start date format")
	}

	var endDatePtr *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return fmt.Errorf("invalid end date format")
		}
		endDatePtr = &endDate
	}

	status := req.Status
	if status == "" {
		status = existing.Status
	}

	updated := &domain.Contract{
		CustomerContactName:  req.CustomerContactName,
		CustomerContactPhone: req.CustomerContactPhone,
		CustomerContactEmail: req.CustomerContactEmail,
		ProjectName:          req.ProjectName,
		ContractNo:           req.ContractNo,
		StartDate:            startDate,
		EndDate:              endDatePtr,
		Status:               status,
	}
	updated.ID = id
	updated.CreatedAt = existing.CreatedAt

	if err := s.contractRepo.Update(updated, modifiedBy); err != nil {
		return err
	}

	s.auditService.CreateAuditLog("Contract", id, domain.AuditActionUpdate, existing, updated, modifiedBy)
	return nil
}

func (s *contractService) DeleteContract(id uint, deletedBy string) error {
	existing, err := s.contractRepo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.contractRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	s.auditService.CreateAuditLog("Contract", id, domain.AuditActionDelete, existing, nil, deletedBy)
	return nil
}
