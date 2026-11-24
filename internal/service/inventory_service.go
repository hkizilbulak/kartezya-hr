package service

import (
	"errors"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type InventoryService interface {
	CreateItem(item *domain.InventoryItem, createdBy string) error
	UpdateItem(item *domain.InventoryItem, modifiedBy string) error
	DeleteItem(id uint, deletedBy string) error
	GetItemByID(id uint) (*domain.InventoryItem, error)
	GetItemsByEmployeeID(employeeID uint) ([]*domain.InventoryItem, error)
	GetItemsReport(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.InventoryItem, int64, error)
	GetItemHistories(inventoryItemID uint) ([]*domain.InventoryAssignmentHistory, error)
}

type inventoryService struct {
	inventoryRepo repository.InventoryRepository
	employeeRepo  repository.EmployeeRepository
	auditService  AuditService
}

func NewInventoryService(
	inventoryRepo repository.InventoryRepository,
	employeeRepo repository.EmployeeRepository,
	auditService AuditService,
) InventoryService {
	return &inventoryService{
		inventoryRepo: inventoryRepo,
		employeeRepo:  employeeRepo,
		auditService:  auditService,
	}
}

func (s *inventoryService) validateItem(item *domain.InventoryItem) error {
	if item.Status == domain.InventoryStatusInUse {
		if item.EmployeeID == nil {
			return errors.New("employee ID is required when status is IN_USE")
		}
		if item.AssignmentDate == nil {
			now := time.Now()
			item.AssignmentDate = &now
		}
	} else {
		// If not in use, should not have an employee assigned
		item.EmployeeID = nil
		item.AssignmentDate = nil
	}
	return nil
}

func (s *inventoryService) CreateItem(item *domain.InventoryItem, createdBy string) error {
	if err := s.validateItem(item); err != nil {
		return err
	}

	err := s.inventoryRepo.Transaction(func(txRepo repository.InventoryRepository) error {
		if err := txRepo.Create(item, createdBy); err != nil {
			return err
		}

		if item.Status == domain.InventoryStatusInUse && item.EmployeeID != nil {
			asgnDate := item.AssignmentDate
			if asgnDate == nil {
				now := time.Now()
				asgnDate = &now
			}
			history := domain.InventoryAssignmentHistory{
				InventoryItemID: item.ID,
				EmployeeID:      item.EmployeeID,
				Status:          item.Status,
				Action:          "ASSIGNED",
				AssignmentDate:  asgnDate,
				Notes:           item.Notes,
			}
			if err := txRepo.CreateHistory(&history, createdBy); err != nil {
				return err
			}
		}
		return nil
	})

	if err == nil {
		s.auditService.CreateAuditLog("InventoryItem", item.ID, "CREATE", nil, item, createdBy)
	}
	return err
}

func (s *inventoryService) UpdateItem(item *domain.InventoryItem, modifiedBy string) error {
	existing, err := s.inventoryRepo.GetByID(item.ID)
	if err != nil {
		return err
	}

	prevStatus := existing.Status
	prevEmpID := existing.EmployeeID

	if err := s.validateItem(item); err != nil {
		return err
	}

	err = s.inventoryRepo.Transaction(func(txRepo repository.InventoryRepository) error {
		now := time.Now()

		wasInUse := prevStatus == domain.InventoryStatusInUse && prevEmpID != nil
		nowInUse := item.Status == domain.InventoryStatusInUse && item.EmployeeID != nil

		switch {
		case wasInUse && !nowInUse:
			// Item returned / unassigned
			action := "RETURNED"
			if item.Status == domain.InventoryStatusDamaged {
				action = "DAMAGED"
			}
			active, _ := txRepo.GetActiveHistoryByItemID(item.ID)
			if active != nil {
				if err := txRepo.CloseActiveHistory(item.ID, &now, item.Status, action, item.Notes, modifiedBy); err != nil {
					return err
				}
			} else {
				// Fallback: If no active history exists (e.g. legacy item), create a closed history record
				asgnDate := existing.AssignmentDate
				if asgnDate == nil {
					asgnDate = &existing.CreatedAt
				}
				closedHist := domain.InventoryAssignmentHistory{
					InventoryItemID: item.ID,
					EmployeeID:      prevEmpID,
					Status:          item.Status,
					Action:          action,
					AssignmentDate:  asgnDate,
					ReturnDate:      &now,
					Notes:           item.Notes,
				}
				if err := txRepo.CreateHistory(&closedHist, modifiedBy); err != nil {
					return err
				}
			}

		case !wasInUse && nowInUse:
			// Newly assigned to an employee
			history := domain.InventoryAssignmentHistory{
				InventoryItemID: item.ID,
				EmployeeID:      item.EmployeeID,
				Status:          item.Status,
				Action:          "ASSIGNED",
				AssignmentDate:  item.AssignmentDate,
				Notes:           item.Notes,
			}
			if err := txRepo.CreateHistory(&history, modifiedBy); err != nil {
				return err
			}

		case wasInUse && nowInUse && *prevEmpID != *item.EmployeeID:
			// Direct transfer between employees
			if err := txRepo.CloseActiveHistory(item.ID, &now, item.Status, "TRANSFERRED", item.Notes, modifiedBy); err != nil {
				return err
			}
			newHist := domain.InventoryAssignmentHistory{
				InventoryItemID: item.ID,
				EmployeeID:      item.EmployeeID,
				Status:          item.Status,
				Action:          "ASSIGNED",
				AssignmentDate:  item.AssignmentDate,
				Notes:           item.Notes,
			}
			if err := txRepo.CreateHistory(&newHist, modifiedBy); err != nil {
				return err
			}
		}

		return txRepo.Update(item, modifiedBy)
	})

	if err == nil {
		s.auditService.CreateAuditLog("InventoryItem", item.ID, "UPDATE", existing, item, modifiedBy)
	}
	return err
}

func (s *inventoryService) DeleteItem(id uint, deletedBy string) error {
	item, err := s.inventoryRepo.GetByID(id)
	if err != nil {
		return err
	}

	err = s.inventoryRepo.Transaction(func(txRepo repository.InventoryRepository) error {
		_ = txRepo.CloseActiveHistory(id, nil, item.Status, "DELETED", "Envanter silindi", deletedBy)
		return txRepo.Delete(id, deletedBy)
	})
	if err == nil {
		s.auditService.CreateAuditLog("InventoryItem", item.ID, "DELETE", item, nil, deletedBy)
	}
	return err
}

func (s *inventoryService) GetItemByID(id uint) (*domain.InventoryItem, error) {
	return s.inventoryRepo.GetByID(id)
}

func (s *inventoryService) GetItemsByEmployeeID(employeeID uint) ([]*domain.InventoryItem, error) {
	return s.inventoryRepo.GetByEmployeeID(employeeID)
}

func (s *inventoryService) GetItemsReport(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.InventoryItem, int64, error) {
	return s.inventoryRepo.GetAllWithFilters(limit, offset, sortParams, filters)
}

func (s *inventoryService) GetItemHistories(inventoryItemID uint) ([]*domain.InventoryAssignmentHistory, error) {
	return s.inventoryRepo.GetHistoriesByItemID(inventoryItemID)
}
