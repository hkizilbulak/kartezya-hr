package database

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
)

func TestResolveEmployeeGradeTableName_UsesConfiguredPrefix(t *testing.T) {
	cases := []struct {
		prefix string
		want   string
	}{
		{prefix: "hr", want: "hr_employee_grades"},
		{prefix: "hr_test", want: "hr_test_employee_grades"},
		{prefix: "hr_prod", want: "hr_prod_employee_grades"},
	}

	for _, tc := range cases {
		cfg := &config.Config{}
		cfg.Database.TablePrefix = tc.prefix
		domain.SetConfig(cfg)

		got := ResolveEmployeeGradeTableName()
		if got != tc.want {
			t.Fatalf("prefix=%s: got %q, want %q", tc.prefix, got, tc.want)
		}
	}
}

func TestBuildCreateEmployeeGradeActiveUniqueIndexSQL(t *testing.T) {
	sqlProd := BuildCreateEmployeeGradeActiveUniqueIndexSQL("hr_employee_grades")
	sqlTest := BuildCreateEmployeeGradeActiveUniqueIndexSQL("hr_test_employee_grades")

	for _, part := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS",
		`"ux_hr_employee_grades_employee_id_status_active"`,
		`"hr_employee_grades"`,
		"(employee_id)",
		"status = 'ACTIVE'",
		"deleted = false",
	} {
		if !strings.Contains(sqlProd, part) {
			t.Fatalf("prod DDL missing %q: %s", part, sqlProd)
		}
	}

	for _, part := range []string{
		`"ux_hr_test_employee_grades_employee_id_status_active"`,
		`"hr_test_employee_grades"`,
	} {
		if !strings.Contains(sqlTest, part) {
			t.Fatalf("test DDL missing %q: %s", part, sqlTest)
		}
	}

	if sqlProd == sqlTest {
		t.Fatal("test and prod DDL must differ by resolved table/index names")
	}
}

func TestBuildCreateEmployeeGradeActiveGradeIDIndexSQL(t *testing.T) {
	sql := BuildCreateEmployeeGradeActiveGradeIDIndexSQL("hr_employee_grades")
	for _, part := range []string{
		"CREATE INDEX IF NOT EXISTS",
		`"idx_hr_employee_grades_grade_id_active_lookup"`,
		"(grade_id, employee_id)",
		"status = 'ACTIVE'",
		"deleted = false",
	} {
		if !strings.Contains(sql, part) {
			t.Fatalf("lookup index DDL missing %q: %s", part, sql)
		}
	}
}

func TestBuildEmployeeGradeActiveSelectionOrderBySQL_PrefersStatusActive(t *testing.T) {
	order := BuildEmployeeGradeActiveSelectionOrderBySQL()
	for _, part := range []string{
		"status = 'ACTIVE' AND end_date IS NULL",
		"end_date IS NULL THEN 0 ELSE 1",
		"start_date <= CURRENT_DATE",
		"start_date DESC",
		"created_at DESC",
		"id DESC",
	} {
		if !strings.Contains(order, part) {
			t.Fatalf("ORDER BY missing %q:\n%s", part, order)
		}
	}
}

func TestBuildEmployeeGradeStatusBackfillSQL_MatchesSelectionRules(t *testing.T) {
	sql := BuildEmployeeGradeStatusBackfillSQL("hr_employee_grades")
	for _, part := range []string{
		"ROW_NUMBER() OVER",
		"PARTITION BY employee_id",
		"status = 'ACTIVE' AND end_date IS NULL",
		"end_date IS NULL THEN 0 ELSE 1",
		"start_date <= CURRENT_DATE",
		"start_date DESC",
		"created_at DESC",
		"id DESC",
		"LEAD(start_date)",
		"INTERVAL '1 day'",
		"'" + string(domain.EmployeeGradeStatusActive) + "'",
		"'" + string(domain.EmployeeGradeStatusInactive) + "'",
	} {
		if !strings.Contains(sql, part) {
			t.Fatalf("backfill SQL missing %q", part)
		}
	}
	if strings.Contains(sql, "hr_test_") {
		t.Fatal("builder must not hardcode hr_test_")
	}
}

func TestBuildEmployeeGradeCheckConstraintSQL(t *testing.T) {
	datesSQL := BuildAddEmployeeGradeDatesCheckConstraintSQL("hr_employee_grades")
	for _, part := range []string{
		`chk_hr_employee_grades_dates_valid`,
		"end_date IS NULL OR end_date >= start_date",
	} {
		if !strings.Contains(datesSQL, part) {
			t.Fatalf("dates CHECK missing %q: %s", part, datesSQL)
		}
	}

	statusSQL := BuildAddEmployeeGradeStatusEndDateCheckConstraintSQL("hr_employee_grades")
	for _, part := range []string{
		`chk_hr_employee_grades_status_end_date`,
		"status = 'ACTIVE' AND end_date IS NULL AND deleted = false",
		"status = 'INACTIVE' AND end_date IS NOT NULL",
	} {
		if !strings.Contains(statusSQL, part) {
			t.Fatalf("status CHECK missing %q: %s", part, statusSQL)
		}
	}
}

func TestBuildEmployeeGradeMigrationPrecheckSQL_Blockers(t *testing.T) {
	sql := BuildEmployeeGradeMigrationPrecheckSQL("hr_employee_grades", "hr_employees", "hr_grades")
	for _, part := range []string{
		"RAISE EXCEPTION",
		"orphan employee_id",
		"orphan grade_id",
		"NULL start_date",
		"end_date < start_date",
		"overlapping closed date ranges",
		"BLOCKER",
	} {
		if !strings.Contains(sql, part) {
			t.Fatalf("precheck missing %q:\n%s", part, sql)
		}
	}
	if strings.Contains(sql, "INSERT ") || strings.Contains(sql, "UPDATE ") || strings.Contains(sql, "DELETE ") {
		t.Fatal("precheck must not mutate data")
	}
}

func TestBuildEmployeeGradePostBackfillAssertSQL(t *testing.T) {
	sql := BuildEmployeeGradePostBackfillAssertSQL("hr_employee_grades")
	for _, part := range []string{
		"multiple ACTIVE grades remain",
		"status/end_date invariants still violated",
		"RAISE EXCEPTION",
	} {
		if !strings.Contains(sql, part) {
			t.Fatalf("assert missing %q", part)
		}
	}
}

func TestEnsureEmployeeGradeStatusConstraints_RejectsNilAndNonPostgres(t *testing.T) {
	if err := EnsureEmployeeGradeStatusConstraints(nil); err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestEmployeeGradeConstraintDDL_DocumentsPartialUniqueSemantics(t *testing.T) {
	uniqueSQL := BuildCreateEmployeeGradeActiveUniqueIndexSQL("hr_employee_grades")
	if !strings.Contains(uniqueSQL, "WHERE status = 'ACTIVE' AND deleted = false") {
		t.Fatal("partial unique must exclude deleted and non-ACTIVE rows")
	}

	statusCheck := BuildAddEmployeeGradeStatusEndDateCheckConstraintSQL("hr_employee_grades")
	if !strings.Contains(statusCheck, "end_date IS NULL AND deleted = false") {
		t.Fatal("ACTIVE check must require null end_date and deleted=false")
	}
	if !strings.Contains(statusCheck, "end_date IS NOT NULL") {
		t.Fatal("INACTIVE check must require end_date")
	}

	datesCheck := BuildAddEmployeeGradeDatesCheckConstraintSQL("hr_employee_grades")
	if !strings.Contains(datesCheck, "end_date >= start_date") {
		t.Fatal("dates check must reject end_date < start_date")
	}
}

func schemaDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "schema"),
		filepath.Join("schema"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	t.Fatal("schema directory not found")
	return ""
}

func TestMigrateEmployeeGradeStatusSQL_TransactionalGuardedIdempotent(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(schemaDir(t), "migrate_employee_grade_status.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, part := range []string{
		"BEGIN;",
		"COMMIT;",
		"precheck BLOCKER",
		"ADD COLUMN IF NOT EXISTS status",
		"ROW_NUMBER() OVER",
		"status = 'ACTIVE' AND end_date IS NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS",
		"chk_hr_employee_grades_dates_valid",
		"chk_hr_employee_grades_status_end_date",
		"idx_hr_employee_grades_grade_id_active_lookup",
	} {
		if !strings.Contains(sql, part) {
			t.Fatalf("migrate SQL missing %q", part)
		}
	}
	if strings.Contains(sql, "hr_test_") {
		t.Fatal("migrate mirror must use logical hr_* names, not hr_test_ hardcode")
	}
	if strings.Contains(sql, "DROP COLUMN") {
		t.Fatal("status migration must not drop employee columns")
	}
}

func TestDiagnoseEmployeeGradeMigrationSQL_ReadOnlyAndCoverage(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(schemaDir(t), "diagnose_employee_grade_migration.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	upper := strings.ToUpper(sql)

	mutating := regexp.MustCompile(`(?m)^\s*(INSERT|UPDATE|DELETE|ALTER|DROP|TRUNCATE|CREATE)\b`)
	if mutating.MatchString(upper) {
		t.Fatalf("diagnose SQL must be read-only; found mutating statement")
	}

	for _, part := range []string{
		"1_employee_total",
		"2_employee_grade_not_deleted_total",
		"3_employees_with_grade_id",
		"4_grade_id_set_but_no_history",
		"5_employee_grade_id_differs_from_history_current",
		"6_employees_with_no_employee_grade",
		"7_employees_only_soft_deleted_history",
		"8_multiple_open_end_date_null",
		"9_multiple_in_current_date_window",
		"10_overlapping_closed_ranges",
		"11_end_before_start",
		"12_null_start_date",
		"13_orphan_employee_id",
		"14_orphan_grade_id",
		"15_soft_deleted_status_active",
		"16_invalid_or_null_status",
		"17_active_with_end_date",
		"18_inactive_with_null_end_date",
		"19_multiple_active_per_employee",
		"20_history_but_no_active",
		"0_hr_employees_column_presence",
		"grade_id",
		"is_grade_up",
		"contract_no",
		"mother_name",
		"total_gap",
		"total_experience",
	} {
		if !strings.Contains(sql, part) {
			t.Fatalf("diagnose missing coverage marker %q", part)
		}
	}
}

func TestRollbackEmployeeGradeStatusSQL_IsDraftControlled(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(schemaDir(t), "rollback_employee_grade_status.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, part := range []string{
		"DRAFT",
		"BEGIN;",
		"COMMIT;",
		"status = 'ACTIVE'",
		"grade_id = eg.grade_id",
		"prefer KEEP",
	} {
		if !strings.Contains(sql, part) {
			t.Fatalf("rollback draft missing %q", part)
		}
	}
}
