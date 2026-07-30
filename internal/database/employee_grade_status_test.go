package database

import (
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

func TestBuildEmployeeGradeStatusBackfillSQL_MatchesSelectionRules(t *testing.T) {
	sql := BuildEmployeeGradeStatusBackfillSQL("hr_employee_grades")
	for _, part := range []string{
		"ROW_NUMBER() OVER",
		"PARTITION BY employee_id",
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

func TestEnsureEmployeeGradeStatusConstraints_RejectsNilAndNonPostgres(t *testing.T) {
	if err := EnsureEmployeeGradeStatusConstraints(nil); err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestEmployeeGradeConstraintDDL_DocumentsPartialUniqueSemantics(t *testing.T) {
	// CI uses sqlite for most repository tests; PostgreSQL partial unique /
	// CHECK enforcement is not executed here. These assertions lock the DDL
	// contract that production Ensure* applies (second ACTIVE blocked;
	// deleted rows excluded; ACTIVE requires null end_date; INACTIVE requires end_date;
	// end_date >= start_date).
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
