package repository

import (
	"fmt"
	"testing"
	"time"

	"kartezya-hr/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newEmployeeContractTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}

	if err := db.AutoMigrate(&domain.Contract{}, &domain.EmployeeContract{}); err != nil {
		t.Fatalf("failed to migrate test schema: %v", err)
	}

	return db
}

func seedContract(t *testing.T, db *gorm.DB, id uint) {
	t.Helper()

	contract := &domain.Contract{
		AuditableModel:      domain.AuditableModel{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerContactName: "Customer",
		ProjectName:         "Project",
		ContractNo:          fmt.Sprintf("CN-TEST-%d", id),
		StartDate:           time.Now(),
		Status:              domain.ContractStatusApproved,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to seed contract: %v", err)
	}
}

func TestEmployeeContractRepositoryGetByContractAndEmployeeIncludingDeleted(t *testing.T) {
	db := newEmployeeContractTestDB(t)
	repo := NewEmployeeContractRepository(db)
	seedContract(t, db, 10)

	record := &domain.EmployeeContract{
		AuditableModel: domain.AuditableModel{Deleted: true},
		ContractID:     10,
		EmployeeID:     3,
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("failed to seed employee contract: %v", err)
	}

	found, err := repo.GetByContractAndEmployeeIncludingDeleted(10, 3)
	if err != nil {
		t.Fatalf("GetByContractAndEmployeeIncludingDeleted returned error: %v", err)
	}
	if found == nil || !found.Deleted {
		t.Fatalf("expected soft-deleted record, got %#v", found)
	}
}

func TestEmployeeContractRepositoryReviveByContractAndEmployee(t *testing.T) {
	db := newEmployeeContractTestDB(t)
	repo := NewEmployeeContractRepository(db)
	seedContract(t, db, 10)

	record := &domain.EmployeeContract{
		AuditableModel: domain.AuditableModel{Deleted: true, CreatedBy: "1", ModifiedBy: "1"},
		ContractID:     10,
		EmployeeID:     3,
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("failed to seed employee contract: %v", err)
	}

	if err := repo.ReviveByContractAndEmployee(10, 3, "7"); err != nil {
		t.Fatalf("ReviveByContractAndEmployee returned error: %v", err)
	}

	found, err := repo.GetByContractAndEmployeeIncludingDeleted(10, 3)
	if err != nil {
		t.Fatalf("GetByContractAndEmployeeIncludingDeleted returned error: %v", err)
	}
	if found == nil {
		t.Fatal("expected revived record to exist")
	}
	if found.Deleted {
		t.Fatal("expected revived record to be active")
	}
	if found.ModifiedBy != "7" {
		t.Fatalf("expected modified_by to be updated, got %q", found.ModifiedBy)
	}
}
