package database

import (
	"fmt"
	"log"
	"strings"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

const jobHistoryReferenceDateUniqueIndexNameSuffix = "job_id_reference_date_active"

// JobHistoryTableLogicalName is the logical table key used with domain.GetTableName.
// Never hardcode environment-specific table names (hr_ / hr_test_); always resolve via prefix config.
const JobHistoryTableLogicalName = "hr_job_history"

// quoteIdent double-quotes a PostgreSQL identifier safely.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// ResolveJobHistoryTableName returns the configured physical table name for job history.
func ResolveJobHistoryTableName() string {
	return domain.GetTableName(JobHistoryTableLogicalName)
}

// BuildJobHistoryReferenceDateUniqueIndexName builds a prefix-aware index name.
func BuildJobHistoryReferenceDateUniqueIndexName(tableName string) string {
	return fmt.Sprintf("ux_%s_%s", tableName, jobHistoryReferenceDateUniqueIndexNameSuffix)
}

// BuildCreateJobHistoryReferenceDateUniqueIndexSQL builds idempotent DDL for the partial unique index.
func BuildCreateJobHistoryReferenceDateUniqueIndexSQL(tableName string) string {
	indexName := BuildJobHistoryReferenceDateUniqueIndexName(tableName)
	return fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (job_id, reference_date) WHERE reference_date IS NOT NULL AND status IN ('RUNNING', 'SUCCESS')`,
		quoteIdent(indexName),
		quoteIdent(tableName),
	)
}

type jobHistoryReferenceDateDuplicate struct {
	JobID         uint
	ReferenceDate string
	Cnt           int64
}

// EnsureJobHistoryReferenceDateUniqueIndex creates a production-safe partial unique index
// that prevents concurrent RUNNING/SUCCESS rows for the same job_id + reference_date.
// Table name is resolved via DB_TABLE_PREFIX / domain.GetTableName (same path for test and production).
func EnsureJobHistoryReferenceDateUniqueIndex(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	dialect := db.Dialector.Name()
	if dialect != "postgres" {
		return fmt.Errorf("job history reference_date unique index requires postgres dialect, got %s", dialect)
	}

	tableName := ResolveJobHistoryTableName()
	if tableName == "" {
		return fmt.Errorf("resolved job history table name is empty")
	}

	if err := assertNoActiveReferenceDateDuplicates(db, tableName); err != nil {
		return err
	}

	sql := BuildCreateJobHistoryReferenceDateUniqueIndexSQL(tableName)
	log.Printf("[Migrate] Ensuring job history reference_date unique index on table %s", tableName)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("failed to create job history reference_date unique index on %s: %w", tableName, err)
	}

	log.Printf("[Migrate] Job history reference_date unique index ensured on %s", tableName)
	return nil
}

func assertNoActiveReferenceDateDuplicates(db *gorm.DB, tableName string) error {
	query := fmt.Sprintf(`
SELECT job_id, reference_date::text AS reference_date, COUNT(*) AS cnt
FROM %s
WHERE reference_date IS NOT NULL
  AND status IN ('RUNNING', 'SUCCESS')
GROUP BY job_id, reference_date
HAVING COUNT(*) > 1
ORDER BY job_id, reference_date
`, quoteIdent(tableName))

	var duplicates []jobHistoryReferenceDateDuplicate
	if err := db.Raw(query).Scan(&duplicates).Error; err != nil {
		return fmt.Errorf("failed to check duplicate job history reference_date rows on %s: %w", tableName, err)
	}

	if len(duplicates) == 0 {
		return nil
	}

	sampleLimit := 5
	if len(duplicates) < sampleLimit {
		sampleLimit = len(duplicates)
	}
	samples := make([]string, 0, sampleLimit)
	for i := 0; i < sampleLimit; i++ {
		d := duplicates[i]
		samples = append(samples, fmt.Sprintf("job_id=%d reference_date=%s count=%d", d.JobID, d.ReferenceDate, d.Cnt))
	}

	return fmt.Errorf(
		"cannot create unique index on %s: found %d duplicate (job_id, reference_date) combinations with status RUNNING/SUCCESS; clean data manually before deploy. samples: %s",
		tableName,
		len(duplicates),
		strings.Join(samples, "; "),
	)
}
