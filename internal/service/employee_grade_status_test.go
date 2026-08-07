package service

import (
	"testing"
	"time"

	"kartezya-hr/internal/domain"
)

func TestEmployeeGradeStatusFromEndDate_MatchesCreateUpdateContract(t *testing.T) {
	// Request bodies do not accept status; create/update derive it from end_date.
	if domain.EmployeeGradeStatusFromEndDate(nil) != domain.EmployeeGradeStatusActive {
		t.Fatal("open grade must be ACTIVE")
	}
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if domain.EmployeeGradeStatusFromEndDate(&end) != domain.EmployeeGradeStatusInactive {
		t.Fatal("closed grade must be INACTIVE")
	}
}
