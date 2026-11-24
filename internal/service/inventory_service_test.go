package service

import (
	"testing"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type stubInventoryRepo struct {
	items         map[uint]*domain.InventoryItem
	histories     []*domain.InventoryAssignmentHistory
	nextID        uint
	nextHistoryID uint
}

func newStubInventoryRepo() *stubInventoryRepo {
	return &stubInventoryRepo{
		items:         make(map[uint]*domain.InventoryItem),
		histories:     make([]*domain.InventoryAssignmentHistory, 0),
		nextID:        1,
		nextHistoryID: 1,
	}
}

func (s *stubInventoryRepo) Transaction(fn func(txRepo repository.InventoryRepository) error) error {
	return fn(s)
}

func (s *stubInventoryRepo) Create(item *domain.InventoryItem, createdBy string) error {
	item.ID = s.nextID
	s.nextID++
	item.CreatedBy = createdBy
	copied := *item
	s.items[item.ID] = &copied
	return nil
}

func (s *stubInventoryRepo) Update(item *domain.InventoryItem, modifiedBy string) error {
	item.ModifiedBy = modifiedBy
	copied := *item
	s.items[item.ID] = &copied
	return nil
}

func (s *stubInventoryRepo) Delete(id uint, deletedBy string) error {
	if item, ok := s.items[id]; ok {
		item.Deleted = true
		item.ModifiedBy = deletedBy
	}
	return nil
}

func (s *stubInventoryRepo) GetByID(id uint) (*domain.InventoryItem, error) {
	if item, ok := s.items[id]; ok {
		copied := *item
		return &copied, nil
	}
	return nil, nil
}

func (s *stubInventoryRepo) GetByEmployeeID(employeeID uint) ([]*domain.InventoryItem, error) {
	var result []*domain.InventoryItem
	for _, item := range s.items {
		if !item.Deleted && item.EmployeeID != nil && *item.EmployeeID == employeeID && item.Status == domain.InventoryStatusInUse {
			copied := *item
			result = append(result, &copied)
		}
	}
	return result, nil
}

func (s *stubInventoryRepo) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.InventoryItem, int64, error) {
	var result []*domain.InventoryItem
	for _, item := range s.items {
		if !item.Deleted {
			copied := *item
			result = append(result, &copied)
		}
	}
	return result, int64(len(result)), nil
}

func (s *stubInventoryRepo) CreateHistory(history *domain.InventoryAssignmentHistory, createdBy string) error {
	history.ID = s.nextHistoryID
	s.nextHistoryID++
	history.CreatedBy = createdBy
	copied := *history
	s.histories = append(s.histories, &copied)
	return nil
}

func (s *stubInventoryRepo) GetHistoriesByItemID(inventoryItemID uint) ([]*domain.InventoryAssignmentHistory, error) {
	var result []*domain.InventoryAssignmentHistory
	for _, h := range s.histories {
		if h.InventoryItemID == inventoryItemID && !h.Deleted {
			copied := *h
			result = append(result, &copied)
		}
	}
	return result, nil
}

func (s *stubInventoryRepo) GetActiveHistoryByItemID(inventoryItemID uint) (*domain.InventoryAssignmentHistory, error) {
	for i := len(s.histories) - 1; i >= 0; i-- {
		h := s.histories[i]
		if h.InventoryItemID == inventoryItemID && h.ReturnDate == nil && !h.Deleted {
			copied := *h
			return &copied, nil
		}
	}
	return nil, nil
}

func (s *stubInventoryRepo) CloseActiveHistory(inventoryItemID uint, returnDate *time.Time, status domain.InventoryItemStatus, action string, notes string, modifiedBy string) error {
	if returnDate == nil {
		now := time.Now()
		returnDate = &now
	}
	for _, h := range s.histories {
		if h.InventoryItemID == inventoryItemID && h.ReturnDate == nil && !h.Deleted {
			h.ReturnDate = returnDate
			h.Status = status
			h.Action = action
			if notes != "" {
				h.Notes = notes
			}
			h.ModifiedBy = modifiedBy
		}
	}
	return nil
}

type stubAuditService struct{}

func (s *stubAuditService) CreateAuditLog(entityName string, entityID uint, action string, oldValue any, newValue any, performedBy string) error {
	return nil
}

func TestInventoryService_CreateItem_InUse_CreatesHistory(t *testing.T) {
	repo := newStubInventoryRepo()
	svc := NewInventoryService(repo, nil, &stubAuditService{})

	empID := uint(10)
	asgnDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	item := &domain.InventoryItem{
		DeviceType:     "Laptop",
		Brand:          "Apple",
		Model:          "MacBook Pro",
		SerialNumber:   "MBP-1234",
		Status:         domain.InventoryStatusInUse,
		EmployeeID:     &empID,
		AssignmentDate: &asgnDate,
	}

	if err := svc.CreateItem(item, "admin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	histories, err := svc.GetItemHistories(item.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching histories: %v", err)
	}
	if len(histories) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(histories))
	}
	if *histories[0].EmployeeID != empID {
		t.Errorf("expected employee ID %d, got %d", empID, *histories[0].EmployeeID)
	}
	if histories[0].Action != "ASSIGNED" {
		t.Errorf("expected action ASSIGNED, got %s", histories[0].Action)
	}
	if histories[0].ReturnDate != nil {
		t.Errorf("expected nil ReturnDate for active assignment")
	}
}

func TestInventoryService_UpdateItem_ReturnItem_ClosesHistoryAndClearsEmployee(t *testing.T) {
	repo := newStubInventoryRepo()
	svc := NewInventoryService(repo, nil, &stubAuditService{})

	empID := uint(10)
	asgnDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	item := &domain.InventoryItem{
		DeviceType:     "Laptop",
		Brand:          "Apple",
		Model:          "MacBook Pro",
		SerialNumber:   "MBP-1234",
		Status:         domain.InventoryStatusInUse,
		EmployeeID:     &empID,
		AssignmentDate: &asgnDate,
	}
	if err := svc.CreateItem(item, "admin"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Now return the item
	updated := &domain.InventoryItem{
		AuditableModel: domain.AuditableModel{ID: item.ID},
		DeviceType:     item.DeviceType,
		Brand:          item.Brand,
		Model:          item.Model,
		SerialNumber:   item.SerialNumber,
		Status:         domain.InventoryStatusReturned,
		Notes:          "Cihaz temiz iade alındı",
	}

	if err := svc.UpdateItem(updated, "admin"); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// 1. Check item on repository: EmployeeID must be nil
	fetchedItem, err := svc.GetItemByID(item.ID)
	if err != nil {
		t.Fatalf("failed to get item: %v", err)
	}
	if fetchedItem.EmployeeID != nil {
		t.Errorf("expected nil EmployeeID after return, got %v", fetchedItem.EmployeeID)
	}
	if fetchedItem.Status != domain.InventoryStatusReturned {
		t.Errorf("expected status RETURNED, got %s", fetchedItem.Status)
	}

	// 2. Check employee inventory: must NOT show returned item
	empItems, err := svc.GetItemsByEmployeeID(empID)
	if err != nil {
		t.Fatalf("failed to get employee items: %v", err)
	}
	if len(empItems) != 0 {
		t.Errorf("expected 0 active items for employee, got %d", len(empItems))
	}

	// 3. Check history: must have closed history with RETURNED action and ReturnDate set
	histories, err := svc.GetItemHistories(item.ID)
	if err != nil {
		t.Fatalf("failed to get histories: %v", err)
	}
	if len(histories) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(histories))
	}
	if histories[0].ReturnDate == nil {
		t.Errorf("expected non-nil ReturnDate on returned history")
	}
	if histories[0].Action != "RETURNED" {
		t.Errorf("expected action RETURNED, got %s", histories[0].Action)
	}
	if *histories[0].EmployeeID != empID {
		t.Errorf("expected history to remember employee ID %d, got %v", empID, histories[0].EmployeeID)
	}
}

func TestInventoryService_UpdateItem_ReassignToAnotherEmployee(t *testing.T) {
	repo := newStubInventoryRepo()
	svc := NewInventoryService(repo, nil, &stubAuditService{})

	emp1 := uint(10)
	emp2 := uint(20)
	asgnDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	item := &domain.InventoryItem{
		DeviceType:     "Monitor",
		Brand:          "Dell",
		Model:          "U2723QE",
		SerialNumber:   "DELL-9999",
		Status:         domain.InventoryStatusInUse,
		EmployeeID:     &emp1,
		AssignmentDate: &asgnDate,
	}
	if err := svc.CreateItem(item, "admin"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Transfer to emp2
	asgnDate2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	updated := &domain.InventoryItem{
		AuditableModel: domain.AuditableModel{ID: item.ID},
		DeviceType:     item.DeviceType,
		Brand:          item.Brand,
		Model:          item.Model,
		SerialNumber:   item.SerialNumber,
		Status:         domain.InventoryStatusInUse,
		EmployeeID:     &emp2,
		AssignmentDate: &asgnDate2,
		Notes:          "Devir edildi",
	}

	if err := svc.UpdateItem(updated, "admin"); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	histories, err := svc.GetItemHistories(item.ID)
	if err != nil {
		t.Fatalf("failed to get histories: %v", err)
	}
	if len(histories) != 2 {
		t.Fatalf("expected 2 histories, got %d", len(histories))
	}

	// First history (emp1) closed
	h1 := histories[0]
	if *h1.EmployeeID != emp1 || h1.ReturnDate == nil || h1.Action != "TRANSFERRED" {
		t.Errorf("h1 invalid: %+v", h1)
	}

	// Second history (emp2) active
	h2 := histories[1]
	if *h2.EmployeeID != emp2 || h2.ReturnDate != nil || h2.Action != "ASSIGNED" {
		t.Errorf("h2 invalid: %+v", h2)
	}

	// Active inventory checks
	itemsEmp1, _ := svc.GetItemsByEmployeeID(emp1)
	if len(itemsEmp1) != 0 {
		t.Errorf("emp1 should have 0 items, got %d", len(itemsEmp1))
	}
	itemsEmp2, _ := svc.GetItemsByEmployeeID(emp2)
	if len(itemsEmp2) != 1 {
		t.Errorf("emp2 should have 1 item, got %d", len(itemsEmp2))
	}
}

func TestInventoryService_DeleteItem_ClosesHistory(t *testing.T) {
	repo := newStubInventoryRepo()
	svc := NewInventoryService(repo, nil, &stubAuditService{})

	empID := uint(55)
	asgnDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	item := &domain.InventoryItem{
		DeviceType:     "Keyboard",
		Brand:          "Logitech",
		Model:          "MX Keys",
		SerialNumber:   "LOGI-1234",
		Status:         domain.InventoryStatusInUse,
		EmployeeID:     &empID,
		AssignmentDate: &asgnDate,
	}
	if err := svc.CreateItem(item, "admin"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Delete item
	if err := svc.DeleteItem(item.ID, "admin"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Check history: should have DELETED action and return date set
	histories, err := svc.GetItemHistories(item.ID)
	if err != nil {
		t.Fatalf("failed to get histories: %v", err)
	}
	if len(histories) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(histories))
	}
	if histories[0].Action != "DELETED" {
		t.Errorf("expected action DELETED, got %s", histories[0].Action)
	}
	if histories[0].ReturnDate == nil {
		t.Errorf("expected return_date set on deleted history")
	}
}
