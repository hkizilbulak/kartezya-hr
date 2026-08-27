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

	err := s.inventoryRepo.Create(item, createdBy)
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

	if err := s.validateItem(item); err != nil {
		return err
	}

	err = s.inventoryRepo.Update(item, modifiedBy)
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

	err = s.inventoryRepo.Delete(id, deletedBy)
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
