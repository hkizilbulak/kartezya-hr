package repository

import (
	"fmt"
	"testing"
	"time"

	"kartezya-hr/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newEmployeeGradeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.EmployeeGrade{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestEmployeeGradeRepository_ExistsAndCloseActive(t *testing.T) {
	db := newEmployeeGradeTestDB(t)
	repo := NewEmployeeGradeRepository(db)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	row := &domain.EmployeeGrade{
		EmployeeID: 1,
		GradeID:    2,
		StartDate:  start,
		Status:     domain.EmployeeGradeStatusActive,
	}
	if err := repo.Create(row, "tester"); err != nil {
		t.Fatalf("create: %v", err)
	}

	exists, err := repo.ExistsByEmployeeGradeStartDate(1, 2, start)
	if err != nil || !exists {
		t.Fatalf("expected duplicate exists, err=%v exists=%v", err, exists)
	}

	end := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	if err := repo.CloseActiveAsInactive(row.ID, end, "tester"); err != nil {
		t.Fatalf("close: %v", err)
	}

	active, err := repo.GetActiveByEmployeeIDForUpdate(1)
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active != nil {
		t.Fatalf("expected no ACTIVE after close, got %#v", active)
	}

	got, err := repo.GetByID(row.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.EmployeeGradeStatusInactive || got.EndDate == nil || !got.EndDate.Equal(end) {
		t.Fatalf("unexpected closed row %#v", got)
	}
}

func TestEmployeeGradeRepository_TransactionRollback(t *testing.T) {
	db := newEmployeeGradeTestDB(t)
	repo := NewEmployeeGradeRepository(db)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	row := &domain.EmployeeGrade{
		EmployeeID: 1,
		GradeID:    2,
		StartDate:  start,
		Status:     domain.EmployeeGradeStatusActive,
	}
	if err := repo.Create(row, "tester"); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := repo.Transaction(func(txRepo EmployeeGradeRepository) error {
		end := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
		if err := txRepo.CloseActiveAsInactive(row.ID, end, "tester"); err != nil {
			return err
		}
		return fmt.Errorf("force rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}

	got, err := repo.GetByID(row.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.EmployeeGradeStatusActive || got.EndDate != nil {
		t.Fatalf("row should remain ACTIVE after rollback, got %#v", got)
	}
}
