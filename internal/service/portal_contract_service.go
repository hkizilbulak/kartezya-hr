package service

import (
	"strings"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type PortalContractService interface {
	GetEmployeeContractsStatus(employeeID uint) ([]*types.EmployeePortalContractResponse, error)
}

type portalContractService struct {
	repo         repository.PortalContractRepository
	employeeRepo repository.EmployeeRepository
}

func NewPortalContractService(repo repository.PortalContractRepository, employeeRepo repository.EmployeeRepository) PortalContractService {
	return &portalContractService{
		repo:         repo,
		employeeRepo: employeeRepo,
	}
}

func (s *portalContractService) GetEmployeeContractsStatus(employeeID uint) ([]*types.EmployeePortalContractResponse, error) {
	// First fetch the employee
	employee, err := s.employeeRepo.GetByID(employeeID)
	if err != nil {
		return nil, err
	}

	// Fetch all active portal contracts
	contracts, err := s.repo.GetAllActiveContracts()
	if err != nil {
		return nil, err
	}

	// Fetch custom employee portal contract approvals
	approvals, err := s.repo.GetEmployeeApprovals(employeeID)
	if err != nil {
		return nil, err
	}

	// Fetch UserSetting and KvkkLogs for the employee's user
	userSetting, _ := s.repo.GetUserSettingByUserID(employee.UserID)
	kvkkLogs, _ := s.repo.GetKvkkLogsByUserID(employee.UserID)

	// Helper to find client IP for a document type
	findIP := func(docType string) string {
		for _, log := range kvkkLogs {
			if log.DocumentType == docType && log.ClientIP != "" {
				return log.ClientIP
			}
		}
		return ""
	}

	// Map of custom approvals
	approvalMap := make(map[uint]*domain.EmployeePortalContract)
	for _, approval := range approvals {
		approvalMap[approval.ContractID] = approval
	}

	var response []*types.EmployeePortalContractResponse
	for _, contract := range contracts {
		respItem := &types.EmployeePortalContractResponse{
			ContractID: contract.ID,
			Title:      contract.Title,
			Content:    contract.Content,
			Version:    contract.Version,
			Status:     "pending", // default status
		}

		// Identify if it's one of the 4 basic legal contracts
		titleLower := strings.ToLower(contract.Title)
		isBasicLegal := false

		if userSetting != nil {
			if strings.Contains(titleLower, "aydınlatma") || strings.Contains(titleLower, "kvkk") {
				isBasicLegal = true
				if userSetting.KvkkText == "READ" {
					respItem.Status = "approved"
					respItem.ApprovedAt = userSetting.KvkkTextAt
					respItem.IPAddress = findIP("KVKK_TEXT")
				}
			} else if strings.Contains(titleLower, "gizlilik") {
				isBasicLegal = true
				if userSetting.PrivacyPolicy == "READ" {
					respItem.Status = "approved"
					respItem.ApprovedAt = userSetting.PrivacyPolicyAt
					respItem.IPAddress = findIP("PRIVACY_POLICY")
				}
			} else if strings.Contains(titleLower, "rüşvet") || strings.Contains(titleLower, "yolsuzluk") {
				isBasicLegal = true
				if userSetting.AntiBriberyPolicy == "READ" {
					respItem.Status = "approved"
					respItem.ApprovedAt = userSetting.AntiBriberyPolicyAt
					respItem.IPAddress = findIP("ANTI_BRIBERY_POLICY")
				}
			} else if strings.Contains(titleLower, "fotoğraf") || strings.Contains(titleLower, "görsel") {
				isBasicLegal = true
				if userSetting.PhotoConsent == "APPROVED" {
					respItem.Status = "approved"
					respItem.ApprovedAt = userSetting.PhotoConsentAt
					respItem.IPAddress = findIP("PHOTO_CONSENT")
				} else if userSetting.PhotoConsent == "REJECTED" {
					respItem.Status = "rejected"
					respItem.ApprovedAt = userSetting.PhotoConsentAt
					respItem.IPAddress = findIP("PHOTO_CONSENT")
				}
			}
		}

		// If it's not a basic legal contract or no setting was found, fallback to employee_portal_contracts table
		if !isBasicLegal {
			if approval, exists := approvalMap[contract.ID]; exists {
				respItem.Status = approval.Status
				respItem.ApprovedAt = approval.ApprovedAt
				respItem.IPAddress = approval.IPAddress
			}
		}

		response = append(response, respItem)
	}

	return response, nil
}
