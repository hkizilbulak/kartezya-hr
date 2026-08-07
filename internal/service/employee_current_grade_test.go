package service

import (
	"testing"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

func TestMapCurrentEmployeeGrade_UsesActiveEmployeeGrade(t *testing.T) {
	legacyA := int64(1)
	emp := &domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 10},
		GradeID:        &legacyA, // stale column must be ignored
		CurrentEmployeeGrade: &domain.EmployeeGrade{
			AuditableModel: domain.AuditableModel{ID: 99},
			EmployeeID:     10,
			GradeID:        2,
			StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:         domain.EmployeeGradeStatusActive,
			Grade:          domain.Grade{AuditableModel: domain.AuditableModel{ID: 2}, Name: "B"},
		},
	}

	gid, cur := mapCurrentEmployeeGrade(emp)
	if gid == nil || *gid != 2 {
		t.Fatalf("grade_id = %v, want 2", gid)
	}
	if cur == nil || cur.GradeID != 2 || cur.Grade == nil || cur.Grade.Name != "B" {
		t.Fatalf("current_employee_grade = %#v", cur)
	}
}

func TestMapCurrentEmployeeGrade_NoActiveReturnsNil(t *testing.T) {
	legacyA := int64(1)
	emp := &domain.Employee{
		AuditableModel: domain.AuditableModel{ID: 10},
		GradeID:        &legacyA,
	}
	gid, cur := mapCurrentEmployeeGrade(emp)
	if gid != nil || cur != nil {
		t.Fatalf("expected nils, got %v %#v", gid, cur)
	}
}

func TestMapCurrentEmployeeGrade_IgnoresInactiveAndDeleted(t *testing.T) {
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	inactive := &domain.Employee{
		CurrentEmployeeGrade: &domain.EmployeeGrade{
			AuditableModel: domain.AuditableModel{ID: 1},
			GradeID:        2,
			EndDate:        &end,
			Status:         domain.EmployeeGradeStatusInactive,
		},
	}
	gid, cur := mapCurrentEmployeeGrade(inactive)
	if gid != nil || cur != nil {
		t.Fatal("INACTIVE must not map as current")
	}

	deleted := &domain.Employee{
		CurrentEmployeeGrade: &domain.EmployeeGrade{
			AuditableModel: domain.AuditableModel{ID: 2, Deleted: true},
			GradeID:        2,
			Status:         domain.EmployeeGradeStatusActive,
		},
	}
	gid, cur = mapCurrentEmployeeGrade(deleted)
	if gid != nil || cur != nil {
		t.Fatal("deleted ACTIVE must not map as current")
	}
}

func TestMapCurrentEmployeeGrade_CompatibilityMatchesRelation(t *testing.T) {
	emp := &domain.Employee{
		CurrentEmployeeGrade: &domain.EmployeeGrade{
			AuditableModel: domain.AuditableModel{ID: 5},
			EmployeeID:     1,
			GradeID:        7,
			StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			Status:         domain.EmployeeGradeStatusActive,
		},
	}
	gid, cur := mapCurrentEmployeeGrade(emp)
	if gid == nil || cur == nil || *gid != int64(cur.GradeID) {
		t.Fatalf("compatibility grade_id must equal current_employee_grade.grade_id: %v %#v", gid, cur)
	}
	_ = types.CurrentEmployeeGradeResponse{}
}
