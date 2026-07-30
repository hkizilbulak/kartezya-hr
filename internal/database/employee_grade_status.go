package database

import (
	"fmt"
	"log"
	"strings"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

const (
	employeeGradeStatusActiveUniqueIndexSuffix = "employee_id_status_active"
	employeeGradeActiveGradeIDIndexSuffix      = "grade_id_active_lookup"
	employeeGradeDatesCheckConstraintSuffix    = "dates_valid"
	employeeGradeStatusEndDateCheckSuffix      = "status_end_date"
)

// EmployeeGradeTableLogicalName is the logical table key used with domain.GetTableName.
// Never hardcode environment-specific table names (hr_ / hr_test_); always resolve via prefix config.
const EmployeeGradeTableLogicalName = "hr_employee_grades"

// ResolveEmployeeGradeTableName returns the configured physical table name for employee grades.
func ResolveEmployeeGradeTableName() string {
	return domain.GetTableName(EmployeeGradeTableLogicalName)
}

// BuildEmployeeGradeActiveUniqueIndexName builds a prefix-aware partial unique index name.
func BuildEmployeeGradeActiveUniqueIndexName(tableName string) string {
	return fmt.Sprintf("ux_%s_%s", tableName, employeeGradeStatusActiveUniqueIndexSuffix)
}

// BuildEmployeeGradeDatesCheckConstraintName builds a prefix-aware CHECK name for date order.
func BuildEmployeeGradeDatesCheckConstraintName(tableName string) string {
	return fmt.Sprintf("chk_%s_%s", tableName, employeeGradeDatesCheckConstraintSuffix)
}

// BuildEmployeeGradeStatusEndDateCheckConstraintName builds a prefix-aware CHECK name
// for ACTIVE/INACTIVE ↔ end_date invariants.
func BuildEmployeeGradeStatusEndDateCheckConstraintName(tableName string) string {
	return fmt.Sprintf("chk_%s_%s", tableName, employeeGradeStatusEndDateCheckSuffix)
}

// BuildCreateEmployeeGradeActiveUniqueIndexSQL builds idempotent DDL for:
// one ACTIVE (deleted=false) row per employee_id.
func BuildCreateEmployeeGradeActiveUniqueIndexSQL(tableName string) string {
	indexName := BuildEmployeeGradeActiveUniqueIndexName(tableName)
	return fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (employee_id) WHERE status = '%s' AND deleted = false`,
		quoteIdent(indexName),
		quoteIdent(tableName),
		domain.EmployeeGradeStatusActive,
	)
}

// BuildEmployeeGradeActiveGradeIDIndexName builds a prefix-aware index for
// Employees grade_id filter EXISTS lookups on ACTIVE rows.
func BuildEmployeeGradeActiveGradeIDIndexName(tableName string) string {
	return fmt.Sprintf("idx_%s_%s", tableName, employeeGradeActiveGradeIDIndexSuffix)
}

// BuildCreateEmployeeGradeActiveGradeIDIndexSQL speeds grade_id → employee_id
// lookups used by applyActiveEmployeeGradeIDFilter (EXISTS).
func BuildCreateEmployeeGradeActiveGradeIDIndexSQL(tableName string) string {
	indexName := BuildEmployeeGradeActiveGradeIDIndexName(tableName)
	return fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (grade_id, employee_id) WHERE status = '%s' AND deleted = false`,
		quoteIdent(indexName),
		quoteIdent(tableName),
		domain.EmployeeGradeStatusActive,
	)
}

// BuildAddEmployeeGradeStatusColumnSQL adds status if missing (idempotent).
func BuildAddEmployeeGradeStatusColumnSQL(tableName string) string {
	return fmt.Sprintf(
		`ALTER TABLE %s ADD COLUMN IF NOT EXISTS status VARCHAR(20)`,
		quoteIdent(tableName),
	)
}

// BuildEmployeeGradeStatusBackfillSQL assigns exactly one ACTIVE row per employee
// among deleted=false rows, demotes the rest to INACTIVE, and fills missing end_dates.
//
// Active selection ORDER BY (must stay aligned with domain.SelectActiveEmployeeGradeID):
//
//	end_date IS NULL, in-range for CURRENT_DATE, start_date DESC, created_at DESC, id DESC
//
// INACTIVE end_date fill: next chronological start_date - 1 day when valid, else start_date.
// Soft-deleted rows are forced INACTIVE (never counted by the partial unique index).
//
// employees.grade_id is intentionally not modified in this migration phase.
func BuildEmployeeGradeStatusBackfillSQL(tableName string) string {
	t := quoteIdent(tableName)
	active := string(domain.EmployeeGradeStatusActive)
	inactive := string(domain.EmployeeGradeStatusInactive)

	return fmt.Sprintf(`
WITH ranked AS (
	SELECT
		id,
		ROW_NUMBER() OVER (
			PARTITION BY employee_id
			ORDER BY
				CASE WHEN end_date IS NULL THEN 0 ELSE 1 END,
				CASE
					WHEN start_date <= CURRENT_DATE
						AND (end_date IS NULL OR end_date >= CURRENT_DATE)
					THEN 0 ELSE 1
				END,
				start_date DESC,
				created_at DESC,
				id DESC
		) AS rn
	FROM %s
	WHERE deleted = false
),
with_next AS (
	SELECT
		id,
		start_date,
		LEAD(start_date) OVER (
			PARTITION BY employee_id
			ORDER BY start_date ASC, id ASC
		) AS next_start
	FROM %s
	WHERE deleted = false
),
prepared AS (
	SELECT
		eg.id,
		CASE
			WHEN eg.deleted = true THEN '%s'::text
			WHEN r.rn = 1 THEN '%s'::text
			ELSE '%s'::text
		END AS new_status,
		CASE
			WHEN eg.deleted = true THEN
				COALESCE(
					eg.end_date,
					eg.start_date
				)
			WHEN r.rn = 1 THEN
				NULL::date
			WHEN eg.end_date IS NOT NULL THEN
				eg.end_date
			WHEN n.next_start IS NOT NULL
				AND (n.next_start - INTERVAL '1 day')::date >= eg.start_date THEN
				(n.next_start - INTERVAL '1 day')::date
			ELSE
				eg.start_date
		END AS new_end_date
	FROM %s eg
	LEFT JOIN ranked r ON r.id = eg.id
	LEFT JOIN with_next n ON n.id = eg.id
)
UPDATE %s eg
SET
	status = p.new_status,
	end_date = p.new_end_date
FROM prepared p
WHERE eg.id = p.id
`, t, t, inactive, active, inactive, t, t)
}

// BuildEmployeeGradeStatusNotNullDefaultSQL locks status after backfill.
func BuildEmployeeGradeStatusNotNullDefaultSQL(tableName string) string {
	t := quoteIdent(tableName)
	active := string(domain.EmployeeGradeStatusActive)
	return fmt.Sprintf(`
ALTER TABLE %s ALTER COLUMN status SET DEFAULT '%s';
UPDATE %s SET status = '%s' WHERE status IS NULL;
ALTER TABLE %s ALTER COLUMN status SET NOT NULL
`, t, active, t, active, t)
}

// BuildAddEmployeeGradeDatesCheckConstraintSQL ensures end_date >= start_date when set.
func BuildAddEmployeeGradeDatesCheckConstraintSQL(tableName string) string {
	name := BuildEmployeeGradeDatesCheckConstraintName(tableName)
	return fmt.Sprintf(
		`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s; ALTER TABLE %s ADD CONSTRAINT %s CHECK (end_date IS NULL OR end_date >= start_date)`,
		quoteIdent(tableName),
		quoteIdent(name),
		quoteIdent(tableName),
		quoteIdent(name),
	)
}

// BuildAddEmployeeGradeStatusEndDateCheckConstraintSQL enforces:
// ACTIVE ⇒ end_date NULL and not deleted; INACTIVE ⇒ end_date NOT NULL.
func BuildAddEmployeeGradeStatusEndDateCheckConstraintSQL(tableName string) string {
	name := BuildEmployeeGradeStatusEndDateCheckConstraintName(tableName)
	active := string(domain.EmployeeGradeStatusActive)
	inactive := string(domain.EmployeeGradeStatusInactive)
	return fmt.Sprintf(
		`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s; ALTER TABLE %s ADD CONSTRAINT %s CHECK (
  (status = '%s' AND end_date IS NULL AND deleted = false)
  OR (status = '%s' AND end_date IS NOT NULL)
)`,
		quoteIdent(tableName),
		quoteIdent(name),
		quoteIdent(tableName),
		quoteIdent(name),
		active,
		inactive,
	)
}

type employeeGradeActiveDuplicate struct {
	EmployeeID uint
	Cnt        int64
}

type employeeGradeInvariantViolation struct {
	ID     uint
	Status string
	Reason string
}

// EnsureEmployeeGradeStatusConstraints is the single source of truth for
// employee-grade status column backfill, CHECKs, and the partial unique index.
// Called from Migrate() when DB_AUTO_MIGRATE is enabled. Do not also apply
// schema/migrate_employee_grade_status.sql on the same database.
//
// Does not drop or rewrite employees.grade_id (later phase).
func EnsureEmployeeGradeStatusConstraints(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	dialect := db.Dialector.Name()
	if dialect != "postgres" {
		return fmt.Errorf("employee grade status migration requires postgres dialect, got %s", dialect)
	}

	tableName := ResolveEmployeeGradeTableName()
	if tableName == "" {
		return fmt.Errorf("resolved employee grade table name is empty")
	}

	log.Printf("[Migrate] Ensuring employee grade status model on table %s", tableName)

	if err := db.Exec(BuildAddEmployeeGradeStatusColumnSQL(tableName)).Error; err != nil {
		return fmt.Errorf("failed to add status column on %s: %w", tableName, err)
	}

	if err := db.Exec(BuildEmployeeGradeStatusBackfillSQL(tableName)).Error; err != nil {
		return fmt.Errorf("failed to backfill employee grade status on %s: %w", tableName, err)
	}

	if err := db.Exec(BuildEmployeeGradeStatusNotNullDefaultSQL(tableName)).Error; err != nil {
		return fmt.Errorf("failed to set status NOT NULL on %s: %w", tableName, err)
	}

	if err := assertEmployeeGradeStatusInvariants(db, tableName); err != nil {
		return err
	}

	if err := assertNoDuplicateActiveEmployeeGrades(db, tableName); err != nil {
		return err
	}

	if err := db.Exec(BuildAddEmployeeGradeDatesCheckConstraintSQL(tableName)).Error; err != nil {
		return fmt.Errorf("failed to add dates CHECK on %s: %w", tableName, err)
	}

	if err := db.Exec(BuildAddEmployeeGradeStatusEndDateCheckConstraintSQL(tableName)).Error; err != nil {
		return fmt.Errorf("failed to add status/end_date CHECK on %s: %w", tableName, err)
	}

	if err := db.Exec(BuildCreateEmployeeGradeActiveUniqueIndexSQL(tableName)).Error; err != nil {
		return fmt.Errorf("failed to create active employee grade unique index on %s: %w", tableName, err)
	}

	if err := db.Exec(BuildCreateEmployeeGradeActiveGradeIDIndexSQL(tableName)).Error; err != nil {
		return fmt.Errorf("failed to create active grade_id lookup index on %s: %w", tableName, err)
	}

	log.Printf("[Migrate] Employee grade status constraints ensured on %s", tableName)
	return nil
}

func assertNoDuplicateActiveEmployeeGrades(db *gorm.DB, tableName string) error {
	query := fmt.Sprintf(`
SELECT employee_id, COUNT(*) AS cnt
FROM %s
WHERE status = ? AND deleted = false
GROUP BY employee_id
HAVING COUNT(*) > 1
ORDER BY employee_id
`, quoteIdent(tableName))

	var duplicates []employeeGradeActiveDuplicate
	if err := db.Raw(query, domain.EmployeeGradeStatusActive).Scan(&duplicates).Error; err != nil {
		return fmt.Errorf("failed to check duplicate ACTIVE employee grades on %s: %w", tableName, err)
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
		samples = append(samples, fmt.Sprintf("employee_id=%d count=%d", d.EmployeeID, d.Cnt))
	}
	return fmt.Errorf(
		"cannot create unique index on %s: found %d employees with multiple ACTIVE grades after backfill; clean data manually. samples: %s",
		tableName,
		len(duplicates),
		strings.Join(samples, "; "),
	)
}

func assertEmployeeGradeStatusInvariants(db *gorm.DB, tableName string) error {
	query := fmt.Sprintf(`
SELECT id, status::text AS status,
	CASE
		WHEN end_date IS NOT NULL AND end_date < start_date THEN 'end_date_before_start_date'
		WHEN status = ? AND (end_date IS NOT NULL OR deleted = true) THEN 'active_requires_null_end_date_and_not_deleted'
		WHEN status = ? AND end_date IS NULL THEN 'inactive_requires_end_date'
		WHEN status IS NULL OR status NOT IN (?, ?) THEN 'invalid_status'
		ELSE 'ok'
	END AS reason
FROM %s
WHERE
	(end_date IS NOT NULL AND end_date < start_date)
	OR (status = ? AND (end_date IS NOT NULL OR deleted = true))
	OR (status = ? AND end_date IS NULL)
	OR (status IS NULL OR status NOT IN (?, ?))
ORDER BY id
LIMIT 20
`, quoteIdent(tableName))

	active := string(domain.EmployeeGradeStatusActive)
	inactive := string(domain.EmployeeGradeStatusInactive)

	var violations []employeeGradeInvariantViolation
	if err := db.Raw(
		query,
		active, inactive, active, inactive,
		active, inactive, active, inactive,
	).Scan(&violations).Error; err != nil {
		return fmt.Errorf("failed to validate employee grade status invariants on %s: %w", tableName, err)
	}
	if len(violations) == 0 {
		return nil
	}

	samples := make([]string, 0, len(violations))
	for _, v := range violations {
		samples = append(samples, fmt.Sprintf("id=%d status=%s reason=%s", v.ID, v.Status, v.Reason))
	}
	return fmt.Errorf(
		"employee grade status invariants violated on %s after backfill (%d shown): %s",
		tableName,
		len(violations),
		strings.Join(samples, "; "),
	)
}
