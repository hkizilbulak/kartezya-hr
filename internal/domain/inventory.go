package domain

import (
	"time"
)

// InventoryItemStatus represents the current status of an inventory item.
type InventoryItemStatus string

const (
	InventoryStatusInUse    InventoryItemStatus = "IN_USE"
	InventoryStatusInStock  InventoryItemStatus = "IN_STOCK"
	InventoryStatusDamaged  InventoryItemStatus = "DAMAGED"
	InventoryStatusReturned InventoryItemStatus = "RETURNED"
)

// InventoryItem represents a physical asset assigned to an employee or kept in stock.
type InventoryItem struct {
	AuditableModel

	EmployeeID     *uint               `json:"employee_id" gorm:"index"`
	Employee       *Employee           `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	DeviceType     string              `json:"device_type" gorm:"type:varchar(100);not null"`
	Brand          string              `json:"brand" gorm:"type:varchar(100);not null"`
	Model          string              `json:"model" gorm:"type:varchar(100);not null"`
	SerialNumber   string              `json:"serial_number" gorm:"type:varchar(100);uniqueIndex"`
	Status         InventoryItemStatus `json:"status" gorm:"type:varchar(50);not null;default:'IN_STOCK'"`
	AssignmentDate *time.Time          `json:"assignment_date,omitempty"`
	Notes          string              `json:"notes,omitempty" gorm:"type:text"`
	Specifications string              `json:"specifications,omitempty" gorm:"type:jsonb"`
}

// TableName sets the dynamic table name for InventoryItem
func (InventoryItem) TableName() string {
	return GetTableName("inventory_items")
}
