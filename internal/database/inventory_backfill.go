package database

import (
	"fmt"
	"log"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

// BackfillInventoryHistories creates initial assignment history records for existing inventory items
// that currently have an assigned employee but have no history entries yet.
func BackfillInventoryHistories(db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("database is nil")
	}

	inventoryTable := domain.GetTableName("inventory_items")
	historyTable := domain.GetTableName("inventory_assignment_histories")

	var items []domain.InventoryItem
	err := db.Table(inventoryTable).
		Where("employee_id IS NOT NULL AND deleted = ?", false).
		Where("id NOT IN (SELECT DISTINCT inventory_item_id FROM " + historyTable + " WHERE deleted = false)").
		Find(&items).Error
	if err != nil {
		return 0, fmt.Errorf("failed to check existing inventory items for history backfill: %w", err)
	}

	if len(items) == 0 {
		return 0, nil
	}

	var count int64
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			asgnDate := item.AssignmentDate
			if asgnDate == nil {
				asgnDate = &item.CreatedAt
			}
			history := domain.InventoryAssignmentHistory{
				AuditableModel: domain.AuditableModel{
					CreatedBy: "SYSTEM_BACKFILL",
				},
				InventoryItemID: item.ID,
				EmployeeID:      item.EmployeeID,
				Status:          item.Status,
				Action:          "ASSIGNED",
				AssignmentDate:  asgnDate,
				Notes:           item.Notes,
			}
			if err := tx.Create(&history).Error; err != nil {
				return fmt.Errorf("failed to create backfill history for inventory item %d: %w", item.ID, err)
			}
			count++
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	log.Printf("Backfilled assignment history for %d existing inventory items", count)
	return count, nil
}
