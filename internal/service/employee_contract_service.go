package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type EmployeeContractService interface {
	CreateContract(employeeID uint, contractID uint, createdBy string) (*domain.EmployeeContract, error)
	GetContractByID(id uint) (*types.EmployeeContractResponse, error)
	GetContractsByUserID(userID uint) ([]*types.EmployeeContractWithNames, error)
	UpdateContract(id uint, employeeID uint, contractID uint, modifiedBy string, requestingUserID uint, isAdmin bool) error
	DeleteContract(id uint, deletedBy string) error
	GetAllContracts(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error)
}

type employeeContractService struct {
	contractRepo repository.EmployeeContractRepository
	employeeRepo repository.EmployeeRepository
	auditService AuditService
}

func NewEmployeeContractService(contractRepo repository.EmployeeContractRepository, employeeRepo repository.EmployeeRepository, auditService AuditService) EmployeeContractService {
	return &employeeContractService{
		contractRepo: contractRepo,
		employeeRepo: employeeRepo,
		auditService: auditService,
	}
}

func (s *employeeContractService) CreateContract(employeeID uint, contractID uint, createdBy string) (*domain.EmployeeContract, error) {
	// Check if this employee already has this contract
	exists, err := s.contractRepo.CheckExists(employeeID, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing contract: %v", err)
	}
	if exists {
		return nil, fmt.Errorf("employee already has this contract assigned")
	}

	// Create employee contract
	contract := &domain.EmployeeContract{
		EmployeeID: employeeID,
		ContractID: contractID,
	}

	// Create the contract
	if err := s.contractRepo.Create(contract, createdBy); err != nil {
		return nil, fmt.Errorf("failed to create contract: %v", err)
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("EmployeeContract", contract.ID, domain.AuditActionCreate, nil, contract, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return contract, nil
}

func (s *employeeContractService) GetContractByID(id uint) (*types.EmployeeContractResponse, error) {
	if id == 0 {
		return nil, errors.New("invalid contract ID")
	}

	contract, err := s.contractRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	var contractResp *types.ContractResponse
	if contract.Contract.ID != 0 {
		var startDateStr string
		var endDateStr *string
		if !contract.Contract.StartDate.IsZero() {
			startDateStr = contract.Contract.StartDate.Format(time.RFC3339)
		}
		if contract.Contract.EndDate != nil {
			dateStr := contract.Contract.EndDate.Format(time.RFC3339)
			endDateStr = &dateStr
		}

		contractResp = &types.ContractResponse{
			ID:                   contract.Contract.ID,
			ContractNo:           contract.Contract.ContractNo,
			ProjectName:          contract.Contract.ProjectName,
			CustomerContactName:  contract.Contract.CustomerContactName,
			CustomerContactPhone: contract.Contract.CustomerContactPhone,
			CustomerContactEmail: contract.Contract.CustomerContactEmail,
			StartDate:            startDateStr,
			EndDate:              endDateStr,
			Status:               contract.Contract.Status,
		}
	}

	return &types.EmployeeContractResponse{
		ID:         contract.ID,
		CreatedAt:  contract.CreatedAt,
		UpdatedAt:  contract.UpdatedAt,
		Deleted:    contract.Deleted,
		CreatedBy:  contract.CreatedBy,
		ModifiedBy: contract.ModifiedBy,
		Employee: types.EmployeeLookup{
			ID:        contract.Employee.ID,
			FirstName: contract.Employee.FirstName,
			LastName:  contract.Employee.LastName,
		},
		ContractID: contract.ContractID,
		Contract:   contractResp,
	}, nil
}

func (s *employeeContractService) GetContractsByUserID(userID uint) ([]*types.EmployeeContractWithNames, error) {
	// Get employee record for the user
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("employee not found for user: %v", err)
	}

	// Get contracts for this employee
	contracts, _, err := s.contractRepo.GetByEmployeeID(employee.ID, 1, 100)
	if err != nil {
		return nil, err
	}

	var result []*types.EmployeeContractWithNames

	for _, contract := range contracts {
		var startDateStr string
		var endDateStr *string

		// Check if StartDate is not zero value
		if !contract.Contract.StartDate.IsZero() {
			startDateStr = contract.Contract.StartDate.Format(time.RFC3339)
		}

		if contract.Contract.EndDate != nil {
			dateStr := contract.Contract.EndDate.Format(time.RFC3339)
			endDateStr = &dateStr
		}

		// Determine if contract is active
		isActive := contract.Contract.EndDate == nil || time.Now().Before(*contract.Contract.EndDate)

		// Create DTO with related entity names
		contractDTO := &types.EmployeeContractWithNames{
			ID:           contract.ID,
			EmployeeName: fmt.Sprintf("%s %s", contract.Employee.FirstName, contract.Employee.LastName),
			ContractNo:   contract.Contract.ContractNo,
			StartDate:    startDateStr,
			EndDate:      endDateStr,
			IsActive:     isActive,
		}

		result = append(result, contractDTO)
	}

	return result, nil
}

func (s *employeeContractService) UpdateContract(id uint, employeeID uint, contractID uint, modifiedBy string, requestingUserID uint, isAdmin bool) error {
	// Get existing contract for audit trail
	existingContract, err := s.contractRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Authorization check: Non-admin users can only update their own contracts
	if !isAdmin {
		// Get the employee record associated with the contract to get the UserID
		employee, err := s.employeeRepo.GetByID(existingContract.EmployeeID)
		if err != nil {
			return fmt.Errorf("failed to get employee for authorization: %v", err)
		}

		if employee.UserID != requestingUserID {
			return errors.New("unauthorized to update this contract")
		}
	}

	// Create updated contract object
	contract := &domain.EmployeeContract{
		EmployeeID: employeeID,
		ContractID: contractID,
	}

	// Set the ID after creating the struct
	contract.ID = id

	// Update contract
	if err := s.contractRepo.Update(contract, modifiedBy); err != nil {
		return err
	}

	// Get updated contract for audit
	updatedContract, _ := s.contractRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("EmployeeContract", id, domain.AuditActionUpdate, existingContract, updatedContract, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *employeeContractService) DeleteContract(id uint, deletedBy string) error {
	// Get existing contract for audit trail
	existingContract, err := s.contractRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the contract
	if err := s.contractRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("EmployeeContract", id, domain.AuditActionDelete, existingContract, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *employeeContractService) GetAllContracts(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error) {
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
	var contracts []*domain.EmployeeContract
	var total int64
	var err error

	if employeeID != nil {
		contracts, total, err = s.contractRepo.GetByEmployeeID(*employeeID, page, limit)
	} else {
		contracts, total, err = s.contractRepo.GetAll(limit, offset, sortParams)
	}

	if err != nil {
		return nil, err
	}

	contractResponses := make([]types.EmployeeContractResponse, len(contracts))
	for i, contract := range contracts {
		var contractResp *types.ContractResponse
		if contract.Contract.ID != 0 {
			var startDateStr string
			var endDateStr *string
			if !contract.Contract.StartDate.IsZero() {
				startDateStr = contract.Contract.StartDate.Format(time.RFC3339)
			}
			if contract.Contract.EndDate != nil {
				dateStr := contract.Contract.EndDate.Format(time.RFC3339)
				endDateStr = &dateStr
			}

			contractResp = &types.ContractResponse{
				ID:                   contract.Contract.ID,
				ContractNo:           contract.Contract.ContractNo,
				ProjectName:          contract.Contract.ProjectName,
				CustomerContactName:  contract.Contract.CustomerContactName,
				CustomerContactPhone: contract.Contract.CustomerContactPhone,
				CustomerContactEmail: contract.Contract.CustomerContactEmail,
				StartDate:            startDateStr,
				EndDate:              endDateStr,
				Status:               contract.Contract.Status,
			}
		}

		contractResponses[i] = types.EmployeeContractResponse{
			ID:         contract.ID,
			CreatedAt:  contract.CreatedAt,
			UpdatedAt:  contract.UpdatedAt,
			Deleted:    contract.Deleted,
			CreatedBy:  contract.CreatedBy,
			ModifiedBy: contract.ModifiedBy,
			Employee: types.EmployeeLookup{
				ID:        contract.Employee.ID,
				FirstName: contract.Employee.FirstName,
				LastName:  contract.Employee.LastName,
			},
			ContractID: contract.ContractID,
			Contract:   contractResp,
		}
	}

	return &PaginatedResponse{
		Data: contractResponses,
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
