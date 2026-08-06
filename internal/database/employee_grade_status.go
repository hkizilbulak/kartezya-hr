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

	// Migration-only quarantine schema (not a GORM/domain model). Stable name for idempotent re-runs.
	employeeGradeStatusQuarantineSchema = "employee_grade_status_quarantine"
	employeeGradeOrphanQuarantineReason = "orphan employee_id blocks employee grade status migration"
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

// BuildEmployeeGradeActiveSelectionOrderBySQL is the shared deterministic ORDER BY
// used by status backfill ROW_NUMBER. Must stay aligned with
// domain.SelectActiveEmployeeGradeID.
func BuildEmployeeGradeActiveSelectionOrderBySQL() string {
	active := string(domain.EmployeeGradeStatusActive)
	return fmt.Sprintf(`
				CASE WHEN status = '%s' AND end_date IS NULL THEN 0 ELSE 1 END,
				CASE WHEN end_date IS NULL THEN 0 ELSE 1 END,
				CASE
					WHEN start_date <= CURRENT_DATE
						AND (end_date IS NULL OR end_date >= CURRENT_DATE)
					THEN 0 ELSE 1
				END,
				start_date DESC,
				created_at DESC,
				id DESC`, active)
}

// BuildEmployeeGradeStatusBackfillSQL assigns exactly one ACTIVE row per employee
// among deleted=false rows, demotes the rest to INACTIVE, and fills missing end_dates.
//
// Active selection ORDER BY (must stay aligned with domain.SelectActiveEmployeeGradeID):
//
//	status=ACTIVE+open, end_date IS NULL, in-range for CURRENT_DATE,
//	start_date DESC, created_at DESC, id DESC
//
// INACTIVE end_date fill: next chronological start_date - 1 day when valid, else start_date.
// Soft-deleted rows are forced INACTIVE (never counted by the partial unique index).
//
// employees.grade_id is intentionally not modified in this migration phase.
//
// Executed by ApplyEmployeeGradeStatusMigration when DB_AUTO_MIGRATE=true.
// schema/migrate_employee_grade_status.sql is the ops mirror of the same builders.
func BuildEmployeeGradeStatusBackfillSQL(tableName string) string {
	t := quoteIdent(tableName)
	active := string(domain.EmployeeGradeStatusActive)
	inactive := string(domain.EmployeeGradeStatusInactive)
	orderBy := BuildEmployeeGradeActiveSelectionOrderBySQL()

	return fmt.Sprintf(`
WITH ranked AS (
	SELECT
		id,
		ROW_NUMBER() OVER (
			PARTITION BY employee_id
			ORDER BY %s
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
`, orderBy, t, t, inactive, active, inactive, t, t)
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

// BuildEmployeeGradeOrphanQuarantineTableName returns the migration-only orphan table
// name inside the quarantine schema (e.g. hr_employee_grades_orphan).
func BuildEmployeeGradeOrphanQuarantineTableName(egTable string) string {
	return egTable + "_orphan"
}

// BuildEmployeeGradeOrphanQuarantineSQL moves public employee-grade rows with
// orphan employee_id into a migration-only quarantine schema/table.
//
// Behavior matches the validated company TEST quarantine:
//   - CREATE SCHEMA / TABLE as needed
//   - copy eg.* plus quarantined_at + quarantine_reason (no column loss)
//   - remove copied rows from public (move, not destroy)
//   - does not touch employees.grade_id or related tables
//   - idempotent: second run with public orphan=0 is a no-op; no duplicate ids
//
// Must run AFTER status column exists and BEFORE precheck/backfill.
func BuildEmployeeGradeOrphanQuarantineSQL(egTable, employeesTable string) string {
	schema := employeeGradeStatusQuarantineSchema
	qTable := BuildEmployeeGradeOrphanQuarantineTableName(egTable)
	eg := quoteIdent(egTable)
	emp := quoteIdent(employeesTable)
	sch := quoteIdent(schema)
	qt := quoteIdent(qTable)
	reason := strings.ReplaceAll(employeeGradeOrphanQuarantineReason, `'`, `''`)

	return fmt.Sprintf(`
DO $orphan_quarantine$
DECLARE
	orphan_cnt bigint;
	q_cnt bigint;
BEGIN
	SELECT COUNT(*) INTO orphan_cnt
	FROM %s eg
	LEFT JOIN %s e ON e.id = eg.employee_id
	WHERE e.id IS NULL;

	IF orphan_cnt = 0 THEN
		RETURN;
	END IF;

	CREATE SCHEMA IF NOT EXISTS %s;

	IF to_regclass(%s) IS NULL THEN
		CREATE TABLE %s.%s AS
		SELECT eg.*, NOW()::timestamptz AS quarantined_at, '%s'::text AS quarantine_reason
		FROM %s eg
		LEFT JOIN %s e ON e.id = eg.employee_id
		WHERE e.id IS NULL;
	ELSE
		INSERT INTO %s.%s
		SELECT eg.*, NOW()::timestamptz AS quarantined_at, '%s'::text AS quarantine_reason
		FROM %s eg
		LEFT JOIN %s e ON e.id = eg.employee_id
		WHERE e.id IS NULL
		  AND NOT EXISTS (SELECT 1 FROM %s.%s q WHERE q.id = eg.id);
	END IF;

	DELETE FROM %s eg
	WHERE eg.id IN (
		SELECT eg2.id
		FROM %s eg2
		LEFT JOIN %s e ON e.id = eg2.employee_id
		WHERE e.id IS NULL
	);

	SELECT COUNT(*) INTO orphan_cnt
	FROM %s eg
	LEFT JOIN %s e ON e.id = eg.employee_id
	WHERE e.id IS NULL;
	IF orphan_cnt <> 0 THEN
		RAISE EXCEPTION 'employee_grade_status quarantine: public orphan employee_id remain on %s';
	END IF;

	SELECT COUNT(*) INTO q_cnt FROM %s.%s;
	IF q_cnt < 1 THEN
		RAISE EXCEPTION 'employee_grade_status quarantine: quarantine table empty after move on %s';
	END IF;
END
$orphan_quarantine$;
`,
		eg, emp,
		sch,
		quoteString(schema+"."+qTable),
		sch, qt, reason, eg, emp,
		sch, qt, reason, eg, emp, sch, qt,
		eg, eg, emp,
		eg, emp, egTable,
		sch, qt, egTable,
	)
}

// quoteString returns a single-quoted SQL string literal.
func quoteString(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

// BuildEmployeeGradeMigrationPrecheckSQL returns a PostgreSQL DO block that aborts
// the transaction when critical anomalies exist. Must run AFTER status column exists
// (nullable OK), AFTER orphan quarantine, and BEFORE deterministic backfill.
//
// Blockers:
//   - orphan employee_id (must already be quarantined)
//   - orphan grade_id
//   - NULL start_date
//   - end_date < start_date
//   - closed-interval overlaps (both end_dates set)
//
// Non-blockers (auto-fixed by backfill): multiple open ends, missing status.
func BuildEmployeeGradeMigrationPrecheckSQL(egTable, employeesTable, gradesTable string) string {
	eg := quoteIdent(egTable)
	emp := quoteIdent(employeesTable)
	grades := quoteIdent(gradesTable)

	return fmt.Sprintf(`
DO $precheck$
BEGIN
	-- Orphan employee_id (should be none after quarantine step)
	IF EXISTS (
		SELECT 1 FROM %s eg
		LEFT JOIN %s e ON e.id = eg.employee_id
		WHERE e.id IS NULL
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: orphan employee_id on %s';
	END IF;

	-- Orphan grade_id
	IF EXISTS (
		SELECT 1 FROM %s eg
		LEFT JOIN %s g ON g.id = eg.grade_id
		WHERE g.id IS NULL
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: orphan grade_id on %s';
	END IF;

	-- NULL start_date (defensive; column is normally NOT NULL)
	IF EXISTS (
		SELECT 1 FROM %s WHERE start_date IS NULL
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: NULL start_date on %s';
	END IF;

	-- Invalid dates
	IF EXISTS (
		SELECT 1 FROM %s
		WHERE end_date IS NOT NULL AND end_date < start_date
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: end_date < start_date on %s';
	END IF;

	-- Closed-interval overlaps (both ends set) — ambiguous history, do not auto-fix
	IF EXISTS (
		SELECT 1
		FROM %s a
		JOIN %s b
			ON a.employee_id = b.employee_id
			AND a.id < b.id
			AND a.deleted = false
			AND b.deleted = false
			AND a.end_date IS NOT NULL
			AND b.end_date IS NOT NULL
			AND a.start_date <= b.end_date
			AND b.start_date <= a.end_date
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: overlapping closed date ranges on %s — run diagnose_employee_grade_migration.sql';
	END IF;
END
$precheck$;
`,
		eg, emp, egTable,
		eg, grades, egTable,
		eg, egTable,
		eg, egTable,
		eg, eg, egTable,
	)
}

// Employee grade status migration advisory lock keys (pg_advisory_xact_lock).
// Stable across instances so concurrent Railway boots serialize this migration.
const (
	employeeGradeStatusMigrationLockKey1 int64 = 0x4b415254 // "KART"
	employeeGradeStatusMigrationLockKey2 int64 = 0x45475354 // "EGST"
)

// EnsureEmployeeGradeStatusConstraints is the AutoMigrate hook for the full
// employee-grade status migration (column, backfill, CHECKs, indexes).
// Kept name for call-site stability; implementation is ApplyEmployeeGradeStatusMigration.
func EnsureEmployeeGradeStatusConstraints(db *gorm.DB) error {
	return ApplyEmployeeGradeStatusMigration(db)
}

// ApplyEmployeeGradeStatusMigration runs the validated employee-grade status
// migration in one transaction under a PostgreSQL advisory lock.
//
// Steps match schema/migrate_employee_grade_status.sql (builders are the single
// runtime source of truth; the SQL file is the ops/manual mirror).
//
// Idempotent: safe to re-run; second pass leaves data/constraints equivalent.
// On any error the transaction rolls back and the caller must fail startup.
// Does not modify employees.grade_id. Orphan employee-grade rows are moved to a
// migration-only quarantine schema (not destroyed; related tables untouched).
func ApplyEmployeeGradeStatusMigration(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	dialect := db.Dialector.Name()
	if dialect != "postgres" {
		return fmt.Errorf("employee grade status migration requires postgres dialect, got %s", dialect)
	}

	egTable := ResolveEmployeeGradeTableName()
	if egTable == "" {
		return fmt.Errorf("resolved employee grade table name is empty")
	}
	employeesTable := domain.GetTableName("hr_employees")
	gradesTable := domain.GetTableName("hr_grades")
	if employeesTable == "" || gradesTable == "" {
		return fmt.Errorf("resolved employees/grades table name is empty")
	}

	log.Printf("[Migrate] Applying employee grade status migration on %s (advisory-locked transaction)", egTable)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`SELECT pg_advisory_xact_lock(?, ?)`,
			employeeGradeStatusMigrationLockKey1,
			employeeGradeStatusMigrationLockKey2,
		).Error; err != nil {
			return fmt.Errorf("employee grade status advisory lock failed: %w", err)
		}

		steps := []struct {
			name string
			sql  string
		}{
			{"add_status_column", BuildAddEmployeeGradeStatusColumnSQL(egTable)},
			{"orphan_quarantine", BuildEmployeeGradeOrphanQuarantineSQL(egTable, employeesTable)},
			{"precheck", BuildEmployeeGradeMigrationPrecheckSQL(egTable, employeesTable, gradesTable)},
			{"backfill", BuildEmployeeGradeStatusBackfillSQL(egTable)},
			{"status_default", fmt.Sprintf(
				`ALTER TABLE %s ALTER COLUMN status SET DEFAULT '%s'`,
				quoteIdent(egTable), domain.EmployeeGradeStatusActive,
			)},
			{"status_null_fill", fmt.Sprintf(
				`UPDATE %s SET status = '%s' WHERE status IS NULL`,
				quoteIdent(egTable), domain.EmployeeGradeStatusActive,
			)},
			{"status_not_null", fmt.Sprintf(
				`ALTER TABLE %s ALTER COLUMN status SET NOT NULL`,
				quoteIdent(egTable),
			)},
			{"post_backfill_assert", BuildEmployeeGradePostBackfillAssertSQL(egTable)},
			{"drop_dates_check", fmt.Sprintf(
				`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`,
				quoteIdent(egTable), quoteIdent(BuildEmployeeGradeDatesCheckConstraintName(egTable)),
			)},
			{"add_dates_check", fmt.Sprintf(
				`ALTER TABLE %s ADD CONSTRAINT %s CHECK (end_date IS NULL OR end_date >= start_date)`,
				quoteIdent(egTable), quoteIdent(BuildEmployeeGradeDatesCheckConstraintName(egTable)),
			)},
			{"drop_status_end_check", fmt.Sprintf(
				`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`,
				quoteIdent(egTable), quoteIdent(BuildEmployeeGradeStatusEndDateCheckConstraintName(egTable)),
			)},
			{"add_status_end_check", fmt.Sprintf(
				`ALTER TABLE %s ADD CONSTRAINT %s CHECK (
  (status = '%s' AND end_date IS NULL AND deleted = false)
  OR (status = '%s' AND end_date IS NOT NULL)
)`,
				quoteIdent(egTable),
				quoteIdent(BuildEmployeeGradeStatusEndDateCheckConstraintName(egTable)),
				domain.EmployeeGradeStatusActive,
				domain.EmployeeGradeStatusInactive,
			)},
			{"unique_active_index", BuildCreateEmployeeGradeActiveUniqueIndexSQL(egTable)},
			{"lookup_index", BuildCreateEmployeeGradeActiveGradeIDIndexSQL(egTable)},
			{"post_verify", BuildEmployeeGradePostMigrationVerifySQL(egTable, employeesTable)},
		}

		for _, step := range steps {
			if err := tx.Exec(step.sql).Error; err != nil {
				return fmt.Errorf("employee grade status migration step %s failed: %w", step.name, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("[Migrate] Employee grade status migration applied on %s", egTable)
	return nil
}

// BuildEmployeeGradePostMigrationVerifySQL asserts post-conditions that must hold
// after backfill + constraints (orphan gone from public table, no multi-ACTIVE).
func BuildEmployeeGradePostMigrationVerifySQL(egTable, employeesTable string) string {
	eg := quoteIdent(egTable)
	emp := quoteIdent(employeesTable)
	active := string(domain.EmployeeGradeStatusActive)
	return fmt.Sprintf(`
DO $verify$
BEGIN
	IF EXISTS (
		SELECT 1 FROM %s eg
		LEFT JOIN %s e ON e.id = eg.employee_id
		WHERE e.id IS NULL
	) THEN
		RAISE EXCEPTION 'employee_grade_status verify: orphan employee_id remain on %s';
	END IF;

	IF EXISTS (
		SELECT 1
		FROM %s
		WHERE status = '%s' AND deleted = false
		GROUP BY employee_id
		HAVING COUNT(*) > 1
	) THEN
		RAISE EXCEPTION 'employee_grade_status verify: multiple ACTIVE grades remain on %s';
	END IF;
END
$verify$;
`, eg, emp, egTable, eg, active, egTable)
}

// BuildEmployeeGradePostBackfillAssertSQL fails the transaction if backfill left
// duplicate ACTIVE rows or status/end_date invariant violations.
func BuildEmployeeGradePostBackfillAssertSQL(tableName string) string {
	t := quoteIdent(tableName)
	active := string(domain.EmployeeGradeStatusActive)
	inactive := string(domain.EmployeeGradeStatusInactive)
	return fmt.Sprintf(`
DO $assert$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM %s
		WHERE status = '%s' AND deleted = false
		GROUP BY employee_id
		HAVING COUNT(*) > 1
	) THEN
		RAISE EXCEPTION 'employee_grade_status assert: multiple ACTIVE grades remain on %s';
	END IF;

	IF EXISTS (
		SELECT 1 FROM %s
		WHERE (end_date IS NOT NULL AND end_date < start_date)
		   OR (status = '%s' AND (end_date IS NOT NULL OR deleted = true))
		   OR (status = '%s' AND end_date IS NULL)
		   OR (status IS NULL OR status NOT IN ('%s', '%s'))
	) THEN
		RAISE EXCEPTION 'employee_grade_status assert: status/end_date invariants still violated on %s';
	END IF;
END
$assert$;
`, t, active, tableName, t, active, inactive, active, inactive, tableName)
}
