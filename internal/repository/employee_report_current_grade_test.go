package repository

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGradeReportExperienceSQL_NoTotalGap(t *testing.T) {
	expr := gradeReportExperienceExprSQL()
	for _, want := range []string{
		"AGE(CURRENT_DATE, e.profession_start_date)",
		"EXTRACT(YEAR FROM AGE",
		"EXTRACT(MONTH FROM AGE",
		"/ 12.0",
	} {
		if !strings.Contains(expr, want) {
			t.Fatalf("experience expr missing %q:\n%s", want, expr)
		}
	}
	for _, forbidden := range []string{
		"total_gap",
		"COALESCE",
	} {
		if strings.Contains(expr, forbidden) {
			t.Fatalf("experience expr must not contain %q:\n%s", forbidden, expr)
		}
	}
}

func TestBuildActiveCurrentGradeSelectSQL_UsesActiveNotDateWindow(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.TablePrefix = "hr"
	domain.SetConfig(cfg)

	sql := buildActiveCurrentGradeSelectSQL()
	for _, want := range []string{
		"hr_employee_grades",
		"hr_grades",
		"DISTINCT ON (eg.employee_id)",
		"eg.deleted = false",
		"eg.status = 'ACTIVE'",
		"ORDER BY eg.employee_id, eg.start_date DESC, eg.id DESC",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q:\n%s", want, sql)
		}
	}
	for _, forbidden := range []string{
		"start_date <= CURRENT_DATE",
		"end_date IS NULL OR",
		"end_date >= CURRENT_DATE",
		"hr_test_",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("SQL must not contain %q:\n%s", forbidden, sql)
		}
	}
}

func TestBuildActiveCurrentGradeSelectSQL_PrefixAware(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.TablePrefix = "hr_test"
	domain.SetConfig(cfg)
	defer func() {
		cfg2 := &config.Config{}
		cfg2.Database.TablePrefix = "hr"
		domain.SetConfig(cfg2)
	}()

	sql := buildActiveCurrentGradeSelectSQL()
	if !strings.Contains(sql, "hr_test_employee_grades") || !strings.Contains(sql, "hr_test_grades") {
		t.Fatalf("expected hr_test_ prefixed tables, got:\n%s", sql)
	}
	if strings.Contains(sql, " FROM hr_employee_grades ") || strings.Contains(sql, " JOIN hr_grades ") {
		t.Fatalf("must not hardcode hr_ without test prefix:\n%s", sql)
	}
}

func newReportCurrentGradeTestDB(t *testing.T, prefix string) *gorm.DB {
	t.Helper()
	cfg := &config.Config{}
	cfg.Database.TablePrefix = prefix
	domain.SetConfig(cfg)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Role{},
		&domain.UserRole{},
		&domain.Grade{},
		&domain.Employee{},
		&domain.EmployeeGrade{},
		&domain.EmployeeWorkInformation{},
		&domain.Company{},
		&domain.Department{},
		&domain.JobPosition{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// resolveActiveCurrentGradeNamesSQLite mirrors report current-grade SoT on SQLite
// (no DISTINCT ON). Dirty multi-ACTIVE picks latest start_date, then highest id.
func resolveActiveCurrentGradeNamesSQLite(db *gorm.DB) (map[uint]string, error) {
	egTable := domain.GetTableName("hr_employee_grades")
	gradesTable := domain.GetTableName("hr_grades")
	sql := fmt.Sprintf(`
		SELECT eg.employee_id, g.name AS current_grade
		FROM %s eg
		LEFT JOIN %s g ON g.id = eg.grade_id AND g.deleted = false
		WHERE %s
		  AND eg.id = (
			SELECT eg2.id
			FROM %s eg2
			WHERE eg2.employee_id = eg.employee_id
			  AND %s
			ORDER BY eg2.start_date DESC, eg2.id DESC
			LIMIT 1
		  )
	`, egTable, gradesTable,
		activeEmployeeGradeSQLConditions("eg"),
		egTable,
		activeEmployeeGradeSQLConditions("eg2"),
	)

	type row struct {
		EmployeeID   uint   `gorm:"column:employee_id"`
		CurrentGrade string `gorm:"column:current_grade"`
	}
	var rows []row
	if err := db.Raw(sql).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, r := range rows {
		out[r.EmployeeID] = r.CurrentGrade
	}
	return out, nil
}

func TestReportCurrentGrade_PrefersActiveOverEmployeesGradeID(t *testing.T) {
	db := newReportCurrentGradeTestDB(t, "hr")

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 1}, Email: "a@t.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 1}, Name: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 2}, Name: "B"}).Error; err != nil {
		t.Fatal(err)
	}
	legacyA := int64(1)
	if err := db.Create(&domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 10},
		UserID:         1,
		FirstName:      "E",
		LastName:       "L",
		Status:         "ACTIVE",
		GradeID:        &legacyA,
	}).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 1},
		EmployeeID:     10,
		GradeID:        1,
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        &end,
		Status:         domain.EmployeeGradeStatusInactive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 2},
		EmployeeID:     10,
		GradeID:        2,
		StartDate:      start,
		Status:         domain.EmployeeGradeStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := resolveActiveCurrentGradeNamesSQLite(db)
	if err != nil {
		t.Fatal(err)
	}
	if got[10] != "B" {
		t.Fatalf("current grade = %q, want B (ACTIVE), employees.grade_id was A", got[10])
	}
}

func TestReportCurrentGrade_IgnoresInactiveAndDeleted(t *testing.T) {
	db := newReportCurrentGradeTestDB(t, "hr")

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 1}, Email: "a@t.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 2}, Name: "B"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 10},
		UserID:         1,
		FirstName:      "E",
		LastName:       "L",
		Status:         "ACTIVE",
	}).Error; err != nil {
		t.Fatal(err)
	}
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 1},
		EmployeeID:     10,
		GradeID:        2,
		StartDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        &end,
		Status:         domain.EmployeeGradeStatusInactive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 2, Deleted: true},
		EmployeeID:     10,
		GradeID:        2,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := resolveActiveCurrentGradeNamesSQLite(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[10]; ok {
		t.Fatalf("expected no current grade, got %q", got[10])
	}
}

func TestReportCurrentGrade_DirtyMultiActivePicksDeterministic(t *testing.T) {
	db := newReportCurrentGradeTestDB(t, "hr")

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 1}, Email: "a@t.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 1}, Name: "Older"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 2}, Name: "Newer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 10},
		UserID:         1,
		FirstName:      "E",
		LastName:       "L",
		Status:         "ACTIVE",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 1},
		EmployeeID:     10,
		GradeID:        1,
		StartDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 2},
		EmployeeID:     10,
		GradeID:        2,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := resolveActiveCurrentGradeNamesSQLite(db)
	if err != nil {
		t.Fatal(err)
	}
	if got[10] != "Newer" {
		t.Fatalf("dirty multi-ACTIVE should pick latest start_date, got %q", got[10])
	}
	if len(got) != 1 {
		t.Fatalf("must not duplicate employee rows, got %d", len(got))
	}
}

func TestReportCurrentGrade_ConsistentWithEmployeesGradeFilter(t *testing.T) {
	db := newReportCurrentGradeTestDB(t, "hr")
	repo := NewEmployeeRepository(db)

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 1}, Email: "a@t.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 9}, Name: "G9"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 10},
		UserID:         1,
		FirstName:      "E",
		LastName:       "L",
		Status:         "ACTIVE",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 1},
		EmployeeID:     10,
		GradeID:        9,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := resolveActiveCurrentGradeNamesSQLite(db)
	if err != nil {
		t.Fatal(err)
	}
	if got[10] != "G9" {
		t.Fatalf("report current = %q, want G9", got[10])
	}

	list, total, err := repo.GetAllWithFilters(10, 0, types.SortParams{Sort: "id", Direction: "ASC"}, map[string]interface{}{
		"grade_id": uint(9),
		"status":   "ACTIVE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != 10 {
		t.Fatalf("Employees filter should find same ACTIVE employee, total=%d list=%v", total, list)
	}

	gotCounts := parseGradeCounts(t, mustGradeCounts(t, repo))
	if gotCounts["G9"] < 1 {
		t.Fatalf("dashboard grade count should include G9, got %#v", gotCounts)
	}
}
