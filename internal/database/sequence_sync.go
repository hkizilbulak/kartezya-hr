package database

import (
	"fmt"
	"log"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

// SyncCriticalIDSequences advances Postgres identity/serial sequences to at least
// MAX(id) for tables that may have been restored or migrated with explicit IDs.
// Idempotent and safe to run on every boot; does not modify row data.
func SyncCriticalIDSequences(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("sync sequences: db is nil")
	}
	tables := []string{
		domain.GetTableName("hr_employees"),
		domain.GetTableName("hr_employee_grades"),
	}
	for _, table := range tables {
		if err := syncTableIDSequence(db, table); err != nil {
			return err
		}
	}
	return nil
}

func syncTableIDSequence(db *gorm.DB, tableName string) error {
	sql := buildSyncTableIDSequenceSQL(tableName)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("sync id sequence for %s: %w", tableName, err)
	}
	log.Printf("Synced id sequence for %s", tableName)
	return nil
}

func buildSyncTableIDSequenceSQL(tableName string) string {
	qTable := quoteIdent(tableName)
	return fmt.Sprintf(`
SELECT setval(
  pg_get_serial_sequence(%s, 'id'),
  GREATEST(COALESCE((SELECT MAX(id) FROM %s), 1), 1),
  true
)`, quoteString(tableName), qTable)
}
