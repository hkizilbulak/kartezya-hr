package repository

import (
	"encoding/json"
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

func newEmployeeGradeFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg := &config.Config{}
	cfg.Database.TablePrefix = "hr"
	domain.SetConfig(cfg)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
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

func seedGradeFilterFixture(t *testing.T, db *gorm.DB) {
	t.Helper()

	for _, g := range []domain.Grade{
		{AuditableModel: domain.AuditableModel{ID: 1}, Name: "A"},
		{AuditableModel: domain.AuditableModel{ID: 2}, Name: "B"},
	} {
		g := g
		if err := db.Create(&g).Error; err != nil {
			t.Fatalf("seed grade: %v", err)
		}
	}

	for _, u := range []domain.User{
		{AuditableModel: domain.AuditableModel{ID: 1}, Email: "e1@test.com", Password: "x"},
		{AuditableModel: domain.AuditableModel{ID: 2}, Email: "e2@test.com", Password: "x"},
		{AuditableModel: domain.AuditableModel{ID: 3}, Email: "e3@test.com", Password: "x"},
		{AuditableModel: domain.AuditableModel{ID: 4}, Email: "e4@test.com", Password: "x"},
		{AuditableModel: domain.AuditableModel{ID: 5}, Email: "e5@test.com", Password: "x"},
		{AuditableModel: domain.AuditableModel{ID: 6}, Email: "e6@test.com", Password: "x"},
	} {
		u := u
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	gradeAID := int64(1)
	gradeBID := int64(2)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	for _, e := range []domain.Employee{
		{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 1, FirstName: "E1", LastName: "L", Status: "ACTIVE", GradeID: &gradeAID},
		{AuditableModel: domain.AuditableModel{ID: 2}, UserID: 2, FirstName: "E2", LastName: "L", Status: "ACTIVE"},
		{AuditableModel: domain.AuditableModel{ID: 3}, UserID: 3, FirstName: "E3", LastName: "L", Status: "ACTIVE", GradeID: &gradeBID},
		{AuditableModel: domain.AuditableModel{ID: 4}, UserID: 4, FirstName: "E4", LastName: "L", Status: "ACTIVE"},
		{AuditableModel: domain.AuditableModel{ID: 5, Deleted: true}, UserID: 5, FirstName: "E5", LastName: "L", Status: "ACTIVE"},
		{AuditableModel: domain.AuditableModel{ID: 6}, UserID: 6, FirstName: "E6", LastName: "L", Status: "ACTIVE"},
	} {
		e := e
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("seed employee: %v", err)
		}
	}

	for _, eg := range []domain.EmployeeGrade{
		{AuditableModel: domain.AuditableModel{ID: 1}, EmployeeID: 1, GradeID: 2, StartDate: start, Status: domain.EmployeeGradeStatusActive},
		{AuditableModel: domain.AuditableModel{ID: 2}, EmployeeID: 2, GradeID: 2, StartDate: start, Status: domain.EmployeeGradeStatusActive},
		{AuditableModel: domain.AuditableModel{ID: 3}, EmployeeID: 4, GradeID: 2, StartDate: start, EndDate: &end, Status: domain.EmployeeGradeStatusInactive},
		{AuditableModel: domain.AuditableModel{ID: 4, Deleted: true}, EmployeeID: 5, GradeID: 2, StartDate: start, Status: domain.EmployeeGradeStatusActive},
		{AuditableModel: domain.AuditableModel{ID: 5, Deleted: true}, EmployeeID: 6, GradeID: 2, StartDate: start, EndDate: &end, Status: domain.EmployeeGradeStatusInactive},
	} {
		eg := eg
		if err := db.Create(&eg).Error; err != nil {
			t.Fatalf("seed employee grade: %v", err)
		}
	}
}

func parseGradeCounts(t *testing.T, data []interface{}) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	for _, item := range data {
		b, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var row struct {
			GradeName string `json:"grade_name"`
			Count     int64  `json:"count"`
		}
		if err := json.Unmarshal(b, &row); err != nil {
			t.Fatalf("unmarshal %s: %v", string(b), err)
		}
		out[row.GradeName] = row.Count
	}
	return out
}

func employeeIDSet(emps []*domain.Employee) map[uint]bool {
	out := make(map[uint]bool, len(emps))
	for _, e := range emps {
		out[e.ID] = true
	}
	return out
}

func TestApplyActiveEmployeeGradeIDFilter_IgnoresEmployeesGradeID(t *testing.T) {
	db := newEmployeeGradeFilterTestDB(t)
	seedGradeFilterFixture(t, db)
	repo := NewEmployeeRepository(db)
	sort := types.SortParams{Sort: "id", Direction: "ASC"}

	listA, totalA, err := repo.GetAllWithFilters(100, 0, sort, map[string]interface{}{"grade_id": 1, "status": "ACTIVE"})
	if err != nil {
		t.Fatalf("filter A: %v", err)
	}
	if totalA != 0 || len(listA) != 0 {
		t.Fatalf("grade A filter: want empty, got total=%d list=%v", totalA, employeeIDSet(listA))
	}

	listB, totalB, err := repo.GetAllWithFilters(100, 0, sort, map[string]interface{}{"grade_id": 2, "status": "ACTIVE"})
	if err != nil {
		t.Fatalf("filter B: %v", err)
	}
	ids := employeeIDSet(listB)
	if totalB != 2 || len(listB) != 2 || !ids[1] || !ids[2] {
		t.Fatalf("grade B filter: want {1,2}, got total=%d ids=%v", totalB, ids)
	}

	countOnly, err := repo.GetTotalCountWithFilters(map[string]interface{}{"grade_id": 2, "status": "ACTIVE"})
	if err != nil {
		t.Fatalf("count B: %v", err)
	}
	if countOnly != totalB {
		t.Fatalf("list total %d != GetTotalCountWithFilters %d", totalB, countOnly)
	}
}

func TestApplyActiveEmployeeGradeIDFilter_Pagination(t *testing.T) {
	db := newEmployeeGradeFilterTestDB(t)
	seedGradeFilterFixture(t, db)
	repo := NewEmployeeRepository(db)
	sort := types.SortParams{Sort: "id", Direction: "ASC"}

	page1, total, err := repo.GetAllWithFilters(1, 0, sort, map[string]interface{}{"grade_id": 2, "status": "ACTIVE"})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if total != 2 || len(page1) != 1 {
		t.Fatalf("page1: total=%d len=%d", total, len(page1))
	}

	page2, total2, err := repo.GetAllWithFilters(1, 1, sort, map[string]interface{}{"grade_id": 2, "status": "ACTIVE"})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if total2 != 2 || len(page2) != 1 || page1[0].ID == page2[0].ID {
		t.Fatalf("page2 unexpected: total=%d len=%d ids=%d/%d", total2, len(page2), page1[0].ID, page2[0].ID)
	}
}

func TestApplyActiveEmployeeGradeIDFilter_NoGradeFilterUnchanged(t *testing.T) {
	db := newEmployeeGradeFilterTestDB(t)
	seedGradeFilterFixture(t, db)
	repo := NewEmployeeRepository(db)

	list, total, err := repo.GetAllWithFilters(100, 0, types.SortParams{Sort: "id", Direction: "ASC"}, map[string]interface{}{"status": "ACTIVE"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 5 || len(list) != 5 {
		t.Fatalf("want 5 ACTIVE non-deleted employees, got total=%d len=%d", total, len(list))
	}
}

func TestGetEmployeeCountByGrade_UsesActiveEmployeeGrade(t *testing.T) {
	db := newEmployeeGradeFilterTestDB(t)
	seedGradeFilterFixture(t, db)
	repo := NewEmployeeRepository(db)

	data, err := repo.GetEmployeeCountByGrade()
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	got := parseGradeCounts(t, data)

	if got["B"] != 2 {
		t.Fatalf("grade B count = %d, want 2", got["B"])
	}
	if got["Bilinmiyor"] != 3 {
		t.Fatalf("Bilinmiyor count = %d, want 3", got["Bilinmiyor"])
	}
	if got["A"] != 0 {
		t.Fatalf("unexpected grade A count %d", got["A"])
	}
}

func TestDashboardAndEmployeesGradeFilterParity(t *testing.T) {
	db := newEmployeeGradeFilterTestDB(t)
	seedGradeFilterFixture(t, db)
	repo := NewEmployeeRepository(db)

	_, totalB, err := repo.GetAllWithFilters(100, 0, types.SortParams{Sort: "id", Direction: "ASC"}, map[string]interface{}{
		"grade_id": 2,
		"status":   "ACTIVE",
	})
	if err != nil {
		t.Fatalf("filter: %v", err)
	}

	got := parseGradeCounts(t, mustGradeCounts(t, repo))
	if got["B"] != totalB {
		t.Fatalf("parity fail: dashboard B=%d filter total=%d", got["B"], totalB)
	}
}

func mustGradeCounts(t *testing.T, repo EmployeeRepository) []interface{} {
	t.Helper()
	data, err := repo.GetEmployeeCountByGrade()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestActiveEmployeeGradeScopeHelpersSharePredicate(t *testing.T) {
	join := ActiveEmployeeGradeJoinOn("eg", "hr_employees.id")
	for _, part := range []string{"eg.deleted = false", "eg.status = 'ACTIVE'", "eg.employee_id = hr_employees.id"} {
		if !strings.Contains(join, part) {
			t.Fatalf("join ON missing %q: %s", part, join)
		}
	}

	db := newEmployeeGradeFilterTestDB(t)
	sql := applyActiveEmployeeGradeIDFilter(db.Model(&domain.Employee{}), domain.GetTableName("hr_employees"), uint(2)).
		ToSQL(func(tx *gorm.DB) *gorm.DB {
			var ids []uint
			return tx.Select("id").Find(&ids)
		})
	for _, part := range []string{"EXISTS", "deleted", "ACTIVE", "grade_id"} {
		if !strings.Contains(sql, part) {
			t.Fatalf("EXISTS SQL missing %q: %s", part, sql)
		}
	}
	if strings.Contains(sql, "hr_employees`.`grade_id`") || strings.Contains(sql, "hr_employees.grade_id =") {
		t.Fatalf("filter must not use employees.grade_id: %s", sql)
	}
}

func TestGetEmployeeCountByGrade_DuplicateActiveRowsStillDistinct(t *testing.T) {
	db := newEmployeeGradeFilterTestDB(t)

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 10}, Email: "dup@test.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 9}, Name: "Dup"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 10},
		UserID:         10,
		FirstName:      "Dup",
		LastName:       "E",
		Status:         "ACTIVE",
	}).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, id := range []uint{100, 101} {
		eg := domain.EmployeeGrade{
			AuditableModel: domain.AuditableModel{ID: id},
			EmployeeID:     10,
			GradeID:        9,
			StartDate:      start,
			Status:         domain.EmployeeGradeStatusActive,
		}
		if err := db.Create(&eg).Error; err != nil {
			t.Fatal(err)
		}
	}

	repo := NewEmployeeRepository(db)
	got := parseGradeCounts(t, mustGradeCounts(t, repo))
	if got["Dup"] != 1 {
		t.Fatalf("duplicate ACTIVE rows must count employee once, got %d", got["Dup"])
	}

	list, total, err := repo.GetAllWithFilters(100, 0, types.SortParams{Sort: "id", Direction: "ASC"}, map[string]interface{}{
		"grade_id": 9,
		"status":   "ACTIVE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("EXISTS filter must not duplicate employees: total=%d len=%d", total, len(list))
	}
}
