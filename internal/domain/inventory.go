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

	EmployeeID     *uint                        `json:"employee_id" gorm:"index"`
	Employee       *Employee                    `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	DeviceType     string                       `json:"device_type" gorm:"type:varchar(100);not null"`
	Brand          string                       `json:"brand" gorm:"type:varchar(100);not null"`
	Model          string                       `json:"model" gorm:"type:varchar(100);not null"`
	SerialNumber   string                       `json:"serial_number" gorm:"type:varchar(100);uniqueIndex"`
	Status         InventoryItemStatus          `json:"status" gorm:"type:varchar(50);not null;default:'IN_STOCK'"`
	AssignmentDate *time.Time                   `json:"assignment_date,omitempty"`
	Notes          string                       `json:"notes,omitempty" gorm:"type:text"`
	Specifications string                       `json:"specifications,omitempty" gorm:"type:jsonb"`
	Histories      []InventoryAssignmentHistory `json:"histories,omitempty" gorm:"foreignKey:InventoryItemID"`
}

// TableName sets the dynamic table name for InventoryItem
func (InventoryItem) TableName() string {
	return GetTableName("inventory_items")
}

// InventoryAssignmentHistory tracks the historical assignment lifecycle of an inventory item.
type InventoryAssignmentHistory struct {
	AuditableModel

	InventoryItemID uint                `json:"inventory_item_id" gorm:"not null;index"`
	EmployeeID      *uint               `json:"employee_id,omitempty" gorm:"index"`
	Status          InventoryItemStatus `json:"status" gorm:"type:varchar(50);not null"`
	Action          string              `json:"action" gorm:"type:varchar(50);not null"` // "ASSIGNED", "RETURNED", "TRANSFERRED"
	AssignmentDate  *time.Time          `json:"assignment_date,omitempty"`
	ReturnDate      *time.Time          `json:"return_date,omitempty"`
	Notes           string              `json:"notes,omitempty" gorm:"type:text"`

	// Relationships
	InventoryItem *InventoryItem `json:"inventory_item,omitempty" gorm:"foreignKey:InventoryItemID"`
	Employee      *Employee      `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
}

// TableName sets the dynamic table name for InventoryAssignmentHistory
func (InventoryAssignmentHistory) TableName() string {
	return GetTableName("inventory_assignment_histories")
}
