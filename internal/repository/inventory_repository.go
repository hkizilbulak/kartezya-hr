package repository

import (
	"errors"
	"strings"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type InventoryRepository interface {
	Transaction(fn func(txRepo InventoryRepository) error) error
	Create(item *domain.InventoryItem, createdBy string) error
	Update(item *domain.InventoryItem, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	GetByID(id uint) (*domain.InventoryItem, error)
	GetByEmployeeID(employeeID uint) ([]*domain.InventoryItem, error)
	GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.InventoryItem, int64, error)

	// History operations
	CreateHistory(history *domain.InventoryAssignmentHistory, createdBy string) error
	GetHistoriesByItemID(inventoryItemID uint) ([]*domain.InventoryAssignmentHistory, error)
	GetActiveHistoryByItemID(inventoryItemID uint) (*domain.InventoryAssignmentHistory, error)
	CloseActiveHistory(inventoryItemID uint, returnDate *time.Time, status domain.InventoryItemStatus, action string, notes string, modifiedBy string) error
}

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) Transaction(fn func(txRepo InventoryRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&inventoryRepository{db: tx})
	})
}

func (r *inventoryRepository) Create(item *domain.InventoryItem, createdBy string) error {
	item.CreatedBy = createdBy
	return r.db.Create(item).Error
}

func (r *inventoryRepository) Update(item *domain.InventoryItem, modifiedBy string) error {
	item.ModifiedBy = modifiedBy
	return r.db.Save(item).Error
}

func (r *inventoryRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.InventoryItem{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

func (r *inventoryRepository) GetByID(id uint) (*domain.InventoryItem, error) {
	var item domain.InventoryItem
	err := r.db.Where("id = ? AND deleted = ?", id, false).
		Preload("Employee").
		Preload("Histories", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted = ?", false).Order("created_at DESC, id DESC")
		}).
		Preload("Histories.Employee").
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *inventoryRepository) GetByEmployeeID(employeeID uint) ([]*domain.InventoryItem, error) {
	var items []*domain.InventoryItem
	err := r.db.Where("employee_id = ? AND status = ? AND deleted = ?", employeeID, domain.InventoryStatusInUse, false).
		Preload("Employee").
		Find(&items).Error
	return items, err
}

func (r *inventoryRepository) CreateHistory(history *domain.InventoryAssignmentHistory, createdBy string) error {
	history.CreatedBy = createdBy
	return r.db.Create(history).Error
}

func (r *inventoryRepository) GetHistoriesByItemID(inventoryItemID uint) ([]*domain.InventoryAssignmentHistory, error) {
	var histories []*domain.InventoryAssignmentHistory
	err := r.db.Where("inventory_item_id = ? AND deleted = ?", inventoryItemID, false).
		Order("created_at DESC, id DESC").
		Preload("Employee").
		Find(&histories).Error
	return histories, err
}

func (r *inventoryRepository) GetActiveHistoryByItemID(inventoryItemID uint) (*domain.InventoryAssignmentHistory, error) {
	var history domain.InventoryAssignmentHistory
	err := r.db.Where("inventory_item_id = ? AND return_date IS NULL AND deleted = ?", inventoryItemID, false).
		Order("created_at DESC, id DESC").
		First(&history).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &history, nil
}

func (r *inventoryRepository) CloseActiveHistory(inventoryItemID uint, returnDate *time.Time, status domain.InventoryItemStatus, action string, notes string, modifiedBy string) error {
	if returnDate == nil {
		now := time.Now()
		returnDate = &now
	}
	updates := map[string]interface{}{
		"return_date": returnDate,
		"status":      status,
		"action":      action,
		"modified_by": modifiedBy,
	}
	if notes != "" {
		updates["notes"] = notes
	}
	return r.db.Model(&domain.InventoryAssignmentHistory{}).
		Where("inventory_item_id = ? AND return_date IS NULL AND deleted = ?", inventoryItemID, false).
		Updates(updates).Error
}

func (r *inventoryRepository) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.InventoryItem, int64, error) {
	var items []*domain.InventoryItem
	var total int64

	query := r.db.Model(&domain.InventoryItem{}).Where("deleted = ?", false)

	// Apply filters
	if search, ok := filters["search"].(string); ok && search != "" {
		likePattern := "%" + search + "%"
		query = query.Where("brand ILIKE ? OR model ILIKE ? OR serial_number ILIKE ?", likePattern, likePattern, likePattern)
	}

	if deviceType, ok := filters["device_type"].(string); ok && deviceType != "" {
		deviceTypes := strings.Split(deviceType, ",")
		query = query.Where("device_type IN ?", deviceTypes)
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		statuses := strings.Split(status, ",")
		query = query.Where("status IN ?", statuses)
	}

	if employeeID, ok := filters["employee_id"].(uint); ok && employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	if sortParams.Sort != "" {
		direction := types.NormalizeSortDirection(sortParams.Direction, string(types.ASC))
		query = query.Order(sortParams.Sort + " " + direction)
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Preload("Employee").Preload("Employee.EmployeeWorkInformation").Preload("Employee.EmployeeWorkInformation.Company").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
