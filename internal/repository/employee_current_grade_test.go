package repository

import (
	"fmt"
	"testing"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newEmployeeCurrentGradeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfg := &config.Config{}
	cfg.Database.TablePrefix = "hr"
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
		&domain.Grade{},
		&domain.Employee{},
		&domain.EmployeeGrade{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestEmployeeRepository_GetByID_PreloadsActiveCurrentGrade(t *testing.T) {
	db := newEmployeeCurrentGradeTestDB(t)
	repo := NewEmployeeRepository(db)

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 1}, Email: "a@t.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 2}, Name: "B"}).Error; err != nil {
		t.Fatal(err)
	}
	legacyA := int64(1)
	if err := db.Create(&domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 1},
		UserID:         1,
		FirstName:      "A",
		LastName:       "B",
		Status:         "ACTIVE",
		GradeID:        &legacyA,
	}).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 10},
		EmployeeID:     1,
		GradeID:        2,
		StartDate:      start,
		Status:         domain.EmployeeGradeStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 11},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        &end,
		Status:         domain.EmployeeGradeStatusInactive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.CurrentEmployeeGrade == nil || got.CurrentEmployeeGrade.GradeID != 2 {
		t.Fatalf("expected ACTIVE grade 2, got %#v", got.CurrentEmployeeGrade)
	}
	if got.CurrentEmployeeGrade.Grade.Name != "B" {
		t.Fatalf("expected grade name B, got %#v", got.CurrentEmployeeGrade.Grade)
	}
	// Legacy column still readable from DB but API must not use it as SoT
	if got.GradeID == nil || *got.GradeID != 1 {
		t.Fatalf("fixture legacy grade_id should remain 1 on column, got %v", got.GradeID)
	}
}

func TestEmployeeRepository_Update_DoesNotWriteGradeID(t *testing.T) {
	db := newEmployeeCurrentGradeTestDB(t)
	repo := NewEmployeeRepository(db)

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 1}, Email: "a@t.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	legacy := int64(5)
	if err := db.Create(&domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 1},
		UserID:         1,
		FirstName:      "A",
		LastName:       "B",
		Status:         "ACTIVE",
		GradeID:        &legacy,
	}).Error; err != nil {
		t.Fatal(err)
	}

	newGrade := int64(9)
	emp := &domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 1},
		Email:          "p@t.com",
		FirstName:      "A2",
		LastName:       "B2",
		Status:         "ACTIVE",
		GradeID:        &newGrade, // must be ignored by Updates map
	}
	if err := repo.Update(emp, "tester"); err != nil {
		t.Fatalf("update: %v", err)
	}

	var stored domain.Employee
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatal(err)
	}
	if stored.GradeID == nil || *stored.GradeID != 5 {
		t.Fatalf("employees.grade_id must remain 5, got %v", stored.GradeID)
	}
	if stored.FirstName != "A2" {
		t.Fatalf("other fields should update, first_name=%s", stored.FirstName)
	}
}

func TestEmployeeRepository_Create_IgnoresRequestGradeIDWhenNil(t *testing.T) {
	db := newEmployeeCurrentGradeTestDB(t)
	repo := NewEmployeeRepository(db)

	if err := db.Create(&domain.User{AuditableModel: domain.AuditableModel{ID: 1}, Email: "a@t.com", Password: "x"}).Error; err != nil {
		t.Fatal(err)
	}
	emp := &domain.Employee{
		UserID:    1,
		FirstName: "N",
		LastName:  "E",
		Status:    "ACTIVE",
		// GradeID nil — create must not require grade
	}
	if err := repo.Create(emp, "tester"); err != nil {
		t.Fatalf("create: %v", err)
	}
	var stored domain.Employee
	if err := db.First(&stored, emp.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.GradeID != nil {
		t.Fatalf("expected nil grade_id on create, got %v", *stored.GradeID)
	}
}

func TestEmployeeRepository_List_PreloadsCurrentGradeWithoutDuplicate(t *testing.T) {
	db := newEmployeeCurrentGradeTestDB(t)
	repo := NewEmployeeRepository(db)

	for _, id := range []uint{1, 2} {
		if err := db.Create(&domain.User{
			AuditableModel: domain.AuditableModel{ID: id},
			Email:          fmt.Sprintf("u%d@t.com", id),
			Password:       "x",
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&domain.Employee{
			AuditableModel: domain.AuditableModel{ID: id},
			UserID:         id,
			FirstName:      fmt.Sprintf("E%d", id),
			LastName:       "L",
			Status:         "ACTIVE",
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&domain.Grade{AuditableModel: domain.AuditableModel{ID: 3}, Name: "C"}).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 20},
		EmployeeID:     1,
		GradeID:        3,
		StartDate:      start,
		Status:         domain.EmployeeGradeStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Minimal list path — GetAll also preloads User roles / work info tables may be missing.
	// Use GetByIDs isn't enough; call preload helper directly on Find to assert batch behavior.
	var employees []*domain.Employee
	err := preloadActiveEmployeeGrade(db).
		Where("deleted = ?", false).
		Order("id ASC").
		Find(&employees).Error
	if err != nil {
		t.Fatalf("list preload: %v", err)
	}
	if len(employees) != 2 {
		t.Fatalf("want 2 employees, got %d", len(employees))
	}
	if employees[0].CurrentEmployeeGrade == nil || employees[0].CurrentEmployeeGrade.GradeID != 3 {
		t.Fatalf("employee 1 current grade: %#v", employees[0].CurrentEmployeeGrade)
	}
	if employees[1].CurrentEmployeeGrade != nil {
		t.Fatalf("employee 2 should have no current grade")
	}
	_ = repo
}
