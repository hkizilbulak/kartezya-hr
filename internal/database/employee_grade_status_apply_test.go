package database

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func pgAdminDSN() string {
	if v := os.Getenv("KARTEZYA_PG_ADMIN_DSN"); v != "" {
		return v
	}
	return "postgres://berksayin@127.0.0.1:5432/postgres?sslmode=disable"
}

func openPostgresOrSkip(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	if os.Getenv("KARTEZYA_SKIP_PG_MIGRATION_TEST") == "1" {
		t.Skip("KARTEZYA_SKIP_PG_MIGRATION_TEST=1")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("postgres unavailable (%v); set KARTEZYA_PG_ADMIN_DSN or start local PG", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("postgres db handle: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("postgres ping failed: %v", err)
	}
	return db
}

func withDisposablePostgresDB(t *testing.T, fn func(db *gorm.DB)) {
	t.Helper()
	admin := openPostgresOrSkip(t, pgAdminDSN())
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("admin sql db: %v", err)
	}
	name := fmt.Sprintf("kartezya_eg_mig_ut_%d", time.Now().UnixNano())
	if _, err := adminSQL.Exec(`CREATE DATABASE ` + quoteIdent(name)); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminSQL.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, name)
		_, _ = adminSQL.Exec(`DROP DATABASE IF EXISTS ` + quoteIdent(name))
	})

	dsn := fmt.Sprintf("postgres://berksayin@127.0.0.1:5432/%s?sslmode=disable", name)
	db := openPostgresOrSkip(t, dsn)
	cfg := &config.Config{}
	cfg.Database.TablePrefix = "hr"
	cfg.Database.AutoMigrate = true
	domain.SetConfig(cfg)
	if err := createMinimalEmployeeGradeSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	fn(db)
}

func createMinimalEmployeeGradeSchema(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE hr_employees (
			id BIGSERIAL PRIMARY KEY,
			deleted BOOLEAN NOT NULL DEFAULT false,
			grade_id BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE hr_grades (
			id BIGSERIAL PRIMARY KEY,
			deleted BOOLEAN NOT NULL DEFAULT false,
			name VARCHAR(100) NOT NULL DEFAULT 'G',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE hr_employee_grades (
			id BIGSERIAL PRIMARY KEY,
			employee_id BIGINT NOT NULL,
			grade_id BIGINT NOT NULL,
			start_date DATE NOT NULL,
			end_date DATE,
			deleted BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_by VARCHAR(255),
			modified_by VARCHAR(255)
		)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedBaseGradeAndEmployees(t *testing.T, db *gorm.DB) (gradeID uint, emp1, emp2 uint) {
	t.Helper()
	if err := db.Exec(`INSERT INTO hr_grades (id, name) VALUES (1, 'A')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO hr_employees (id, grade_id) VALUES (10, 1), (20, 1)`).Error; err != nil {
		t.Fatal(err)
	}
	return 1, 10, 20
}

func statusColExists(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	var n int64
	if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema='public' AND table_name='hr_employee_grades' AND column_name='status'`).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func empGradeIDHash(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var h string
	if err := db.Raw(`
SELECT MD5(COALESCE(STRING_AGG(id::text || ':' || COALESCE(grade_id::text,'NULL'), '|' ORDER BY id::text), ''))
FROM hr_employees`).Scan(&h).Error; err != nil {
		t.Fatal(err)
	}
	return h
}

func TestApplyEmployeeGradeStatusMigration_FullAndIdempotent(t *testing.T) {
	withDisposablePostgresDB(t, func(db *gorm.DB) {
		gID, e1, e2 := seedBaseGradeAndEmployees(t, db)
		// e1: two open rows → one ACTIVE after backfill
		if err := db.Exec(`
INSERT INTO hr_employee_grades (employee_id, grade_id, start_date, end_date, deleted, created_at)
VALUES
  (?, ?, '2024-01-01', NULL, false, '2024-01-01'),
  (?, ?, '2025-06-01', NULL, false, '2025-06-01'),
  (?, ?, '2023-01-01', '2023-12-31', false, '2023-01-01')`,
			e1, gID, e1, gID, e2, gID).Error; err != nil {
			t.Fatal(err)
		}
		beforeHash := empGradeIDHash(t, db)

		if err := ApplyEmployeeGradeStatusMigration(db); err != nil {
			t.Fatalf("first apply: %v", err)
		}
		if !statusColExists(t, db) {
			t.Fatal("status column missing after apply")
		}

		var active1 int64
		if err := db.Raw(`SELECT COUNT(*) FROM hr_employee_grades WHERE employee_id=? AND status='ACTIVE' AND deleted=false`, e1).Scan(&active1).Error; err != nil {
			t.Fatal(err)
		}
		if active1 != 1 {
			t.Fatalf("expected 1 ACTIVE for emp1, got %d", active1)
		}

		var multi int64
		if err := db.Raw(`
SELECT COUNT(*) FROM (
  SELECT employee_id FROM hr_employee_grades
  WHERE status='ACTIVE' AND deleted=false
  GROUP BY employee_id HAVING COUNT(*)>1
) s`).Scan(&multi).Error; err != nil {
			t.Fatal(err)
		}
		if multi != 0 {
			t.Fatalf("multi-active=%d", multi)
		}

		var ux, lk int64
		_ = db.Raw(`SELECT COUNT(*) FROM pg_class WHERE relname='ux_hr_employee_grades_employee_id_status_active'`).Scan(&ux)
		_ = db.Raw(`SELECT COUNT(*) FROM pg_class WHERE relname='idx_hr_employee_grades_grade_id_active_lookup'`).Scan(&lk)
		if ux != 1 || lk != 1 {
			t.Fatalf("indexes ux=%d lk=%d", ux, lk)
		}

		if empGradeIDHash(t, db) != beforeHash {
			t.Fatal("employees.grade_id hash changed")
		}

		// Capture row snapshot then second apply
		type row struct {
			ID     uint
			Status string
			End    *time.Time
		}
		var before []row
		if err := db.Raw(`SELECT id, status, end_date AS end FROM hr_employee_grades ORDER BY id`).Scan(&before).Error; err != nil {
			t.Fatal(err)
		}
		if err := ApplyEmployeeGradeStatusMigration(db); err != nil {
			t.Fatalf("second apply: %v", err)
		}
		var after []row
		if err := db.Raw(`SELECT id, status, end_date AS end FROM hr_employee_grades ORDER BY id`).Scan(&after).Error; err != nil {
			t.Fatal(err)
		}
		if len(before) != len(after) {
			t.Fatal("row count changed on second apply")
		}
		for i := range before {
			if before[i].ID != after[i].ID || before[i].Status != after[i].Status {
				t.Fatalf("data drifted on noop apply: before=%+v after=%+v", before[i], after[i])
			}
			if (before[i].End == nil) != (after[i].End == nil) {
				t.Fatalf("end_date nilness drifted id=%d", before[i].ID)
			}
			if before[i].End != nil && after[i].End != nil && !before[i].End.Equal(*after[i].End) {
				t.Fatalf("end_date drifted id=%d", before[i].ID)
			}
		}
		if empGradeIDHash(t, db) != beforeHash {
			t.Fatal("employees.grade_id hash changed on second apply")
		}
	})
}

func TestApplyEmployeeGradeStatusMigration_AutoQuarantinesOrphanIdempotent(t *testing.T) {
	withDisposablePostgresDB(t, func(db *gorm.DB) {
		gID, e1, _ := seedBaseGradeAndEmployees(t, db)
		beforeHash := empGradeIDHash(t, db)
		if err := db.Exec(`
INSERT INTO hr_employee_grades (employee_id, grade_id, start_date, end_date, deleted)
VALUES
  (?, ?, '2024-01-01', NULL, false),
  (99999, ?, '2024-02-01', NULL, false)`, e1, gID, gID).Error; err != nil {
			t.Fatal(err)
		}
		if err := ApplyEmployeeGradeStatusMigration(db); err != nil {
			t.Fatalf("apply with orphan: %v", err)
		}
		var publicOrphan, qCnt, active int64
		_ = db.Raw(`
SELECT COUNT(*) FROM hr_employee_grades eg
LEFT JOIN hr_employees e ON e.id=eg.employee_id WHERE e.id IS NULL`).Scan(&publicOrphan)
		_ = db.Raw(`SELECT COUNT(*) FROM employee_grade_status_quarantine.hr_employee_grades_orphan`).Scan(&qCnt)
		_ = db.Raw(`SELECT COUNT(*) FROM hr_employee_grades WHERE status='ACTIVE' AND deleted=false`).Scan(&active)
		if publicOrphan != 0 || qCnt != 1 || active != 1 {
			t.Fatalf("publicOrphan=%d qCnt=%d active=%d", publicOrphan, qCnt, active)
		}
		if empGradeIDHash(t, db) != beforeHash {
			t.Fatal("employees.grade_id hash changed")
		}
		if err := ApplyEmployeeGradeStatusMigration(db); err != nil {
			t.Fatalf("second apply: %v", err)
		}
		var qCnt2 int64
		_ = db.Raw(`SELECT COUNT(*) FROM employee_grade_status_quarantine.hr_employee_grades_orphan`).Scan(&qCnt2)
		if qCnt2 != 1 {
			t.Fatalf("quarantine duplicate: %d", qCnt2)
		}
	})
}

func TestApplyEmployeeGradeStatusMigration_RejectsOrphanGradeAndRollsBack(t *testing.T) {
	withDisposablePostgresDB(t, func(db *gorm.DB) {
		_, e1, _ := seedBaseGradeAndEmployees(t, db)
		if err := db.Exec(`
INSERT INTO hr_employee_grades (employee_id, grade_id, start_date, end_date, deleted)
VALUES (?, 99999, '2024-01-01', NULL, false)`, e1).Error; err != nil {
			t.Fatal(err)
		}
		err := ApplyEmployeeGradeStatusMigration(db)
		if err == nil {
			t.Fatal("expected orphan grade_id precheck failure")
		}
		if statusColExists(t, db) {
			t.Fatal("status column must not persist after rollback")
		}
		var qSchema int64
		_ = db.Raw(`SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name='employee_grade_status_quarantine'`).Scan(&qSchema)
		if qSchema != 0 {
			t.Fatal("quarantine schema must not persist after rollback")
		}
		var ux int64
		_ = db.Raw(`SELECT COUNT(*) FROM pg_class WHERE relname='ux_hr_employee_grades_employee_id_status_active'`).Scan(&ux)
		if ux != 0 {
			t.Fatal("unique index must not persist after rollback")
		}
	})
}

func TestApplyEmployeeGradeStatusMigration_BlocksSecondActiveInsert(t *testing.T) {
	withDisposablePostgresDB(t, func(db *gorm.DB) {
		gID, e1, _ := seedBaseGradeAndEmployees(t, db)
		if err := db.Exec(`
INSERT INTO hr_employee_grades (employee_id, grade_id, start_date, end_date, deleted)
VALUES (?, ?, '2024-01-01', NULL, false)`, e1, gID).Error; err != nil {
			t.Fatal(err)
		}
		if err := ApplyEmployeeGradeStatusMigration(db); err != nil {
			t.Fatal(err)
		}
		err := db.Exec(`
INSERT INTO hr_employee_grades (employee_id, grade_id, start_date, end_date, deleted, status)
VALUES (?, ?, '2025-01-01', NULL, false, 'ACTIVE')`, e1, gID).Error
		if err == nil {
			t.Fatal("expected unique index to reject second ACTIVE")
		}
	})
}

func TestDatabaseMigrate_AutoMigrateFalseSkipsEmployeeGradeMigration(t *testing.T) {
	withDisposablePostgresDB(t, func(db *gorm.DB) {
		cfg := &config.Config{}
		cfg.Database.TablePrefix = "hr"
		cfg.Database.AutoMigrate = false
		domain.SetConfig(cfg)
		d := &Database{DB: db, Config: cfg}
		if err := d.Migrate(); err != nil {
			t.Fatalf("Migrate with AutoMigrate=false: %v", err)
		}
		if statusColExists(t, db) {
			t.Fatal("DB_AUTO_MIGRATE=false must not add status column")
		}
	})
}

func TestBuildEmployeeGradePostMigrationVerifySQL(t *testing.T) {
	sql := BuildEmployeeGradePostMigrationVerifySQL("hr_employee_grades", "hr_employees")
	for _, part := range []string{"orphan employee_id", "multiple ACTIVE", "RAISE EXCEPTION"} {
		if !strings.Contains(sql, part) {
			t.Fatalf("verify missing %q", part)
		}
	}
}
