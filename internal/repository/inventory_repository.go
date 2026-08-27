package repository

import (
	"strings"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type InventoryRepository interface {
	Create(item *domain.InventoryItem, createdBy string) error
	Update(item *domain.InventoryItem, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	GetByID(id uint) (*domain.InventoryItem, error)
	GetByEmployeeID(employeeID uint) ([]*domain.InventoryItem, error)
	GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.InventoryItem, int64, error)
}

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepository{db: db}
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
	return r.db.Model(&domain.InventoryItem{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": gorm.DeletedAt{Time: r.db.NowFunc(), Valid: true},
	}).Error
}

func (r *inventoryRepository) GetByID(id uint) (*domain.InventoryItem, error) {
	var item domain.InventoryItem
	err := r.db.Preload("Employee").First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *inventoryRepository) GetByEmployeeID(employeeID uint) ([]*domain.InventoryItem, error) {
	var items []*domain.InventoryItem
	err := r.db.Where("employee_id = ?", employeeID).Preload("Employee").Find(&items).Error
	return items, err
}

func (r *inventoryRepository) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.InventoryItem, int64, error) {
	var items []*domain.InventoryItem
	var total int64

	query := r.db.Model(&domain.InventoryItem{})

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
