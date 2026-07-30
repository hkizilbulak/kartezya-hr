package domain

import (
	"testing"
	"time"
)

func TestEmployeeGradeStatusConstants(t *testing.T) {
	if EmployeeGradeStatusActive != "ACTIVE" {
		t.Fatalf("ACTIVE constant = %q", EmployeeGradeStatusActive)
	}
	if EmployeeGradeStatusInactive != "INACTIVE" {
		t.Fatalf("INACTIVE constant = %q", EmployeeGradeStatusInactive)
	}
}

func TestEmployeeGradeStatusFromEndDate(t *testing.T) {
	if got := EmployeeGradeStatusFromEndDate(nil); got != EmployeeGradeStatusActive {
		t.Fatalf("nil end_date => %q, want ACTIVE", got)
	}
	end := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if got := EmployeeGradeStatusFromEndDate(&end); got != EmployeeGradeStatusInactive {
		t.Fatalf("set end_date => %q, want INACTIVE", got)
	}
}

func TestEmployeeGradeModelIncludesStatusField(t *testing.T) {
	eg := EmployeeGrade{
		Status: EmployeeGradeStatusActive,
	}
	if eg.Status != EmployeeGradeStatusActive {
		t.Fatal("EmployeeGrade must expose Status field")
	}
	if eg.TableName() != GetTableName("hr_employee_grades") {
		t.Fatalf("unexpected table name %q", eg.TableName())
	}
}

func TestSelectActiveEmployeeGradeID_PrefersExistingStatusActive(t *testing.T) {
	asOf := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	openNewer := EmployeeGradeActiveCandidate{
		ID: 2, StartDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	markedActiveOlder := EmployeeGradeActiveCandidate{
		ID: 1, StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    EmployeeGradeStatusActive,
	}
	if SelectActiveEmployeeGradeID([]EmployeeGradeActiveCandidate{openNewer, markedActiveOlder}, asOf) != 1 {
		t.Fatal("existing status=ACTIVE must win over newer open unmarked row")
	}
}

func TestSelectActiveEmployeeGradeID_Deterministic(t *testing.T) {
	asOf := time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC)
	closedEnd := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	olderOpen := EmployeeGradeActiveCandidate{
		ID: 1, StartDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	newerOpen := EmployeeGradeActiveCandidate{
		ID: 2, StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	closedInPast := EmployeeGradeActiveCandidate{
		ID: 3, StartDate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: &closedEnd, CreatedAt: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got := SelectActiveEmployeeGradeID([]EmployeeGradeActiveCandidate{closedInPast, olderOpen, newerOpen}, asOf)
	if got != 2 {
		t.Fatalf("expected open row with latest start_date (id=2), got %d", got)
	}

	// Same start_date: higher created_at then higher id
	a := EmployeeGradeActiveCandidate{
		ID: 10, StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	b := EmployeeGradeActiveCandidate{
		ID: 11, StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	if SelectActiveEmployeeGradeID([]EmployeeGradeActiveCandidate{a, b}, asOf) != 11 {
		t.Fatal("tie-break on id DESC failed")
	}

	if SelectActiveEmployeeGradeID(nil, asOf) != 0 {
		t.Fatal("empty candidates should return 0")
	}
}

func TestSelectActiveEmployeeGradeID_PrefersInRangeOverClosedOnlyWhenBothClosed(t *testing.T) {
	asOf := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)
	endInRange := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	endPast := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	inRange := EmployeeGradeActiveCandidate{
		ID: 5, StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: &endInRange, CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	past := EmployeeGradeActiveCandidate{
		ID: 6, StartDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: &endPast, CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if SelectActiveEmployeeGradeID([]EmployeeGradeActiveCandidate{past, inRange}, asOf) != 5 {
		t.Fatal("expected in-range closed row over past-closed row")
	}
}

func TestCloseEndDateForInactive(t *testing.T) {
	start := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	next := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	got := CloseEndDateForInactive(start, &next)
	want := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}

	sameDay := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	got = CloseEndDateForInactive(start, &sameDay)
	if !got.Equal(start) {
		t.Fatalf("same-day next_start must fall back to start_date, got %v", got)
	}

	got = CloseEndDateForInactive(start, nil)
	if !got.Equal(start) {
		t.Fatalf("nil next_start must use start_date, got %v", got)
	}
}

func TestActiveGradeCloseEndDate(t *testing.T) {
	active := time.Date(2026, 1, 1, 15, 30, 0, 0, time.UTC)
	newStart := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	got, err := ActiveGradeCloseEndDate(active, newStart)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}

	_, err = ActiveGradeCloseEndDate(active, active)
	if err != ErrEmployeeGradeInvalidCloseDate {
		t.Fatalf("same-day assign: got %v, want ErrEmployeeGradeInvalidCloseDate", err)
	}

	before := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = ActiveGradeCloseEndDate(active, before)
	if err != ErrEmployeeGradeInvalidCloseDate {
		t.Fatalf("earlier start: got %v, want ErrEmployeeGradeInvalidCloseDate", err)
	}

	nextDay := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	got, err = ActiveGradeCloseEndDate(active, nextDay)
	if err != nil {
		t.Fatalf("next-day assign: %v", err)
	}
	if !got.Equal(dateOnlyUTC(active)) {
		t.Fatalf("next-day close end should equal active start, got %v", got)
	}
}
