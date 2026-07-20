package jobs

import (
	"testing"
	"time"
)

func TestParseReferenceDate_Valid(t *testing.T) {
	got, err := ParseReferenceDate("2026-07-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Format("2006-01-02") != "2026-07-15" {
		t.Fatalf("got %s", got.Format("2006-01-02"))
	}
	if got.Hour() != 0 || got.Minute() != 0 {
		t.Fatalf("expected midnight, got %v", got)
	}
	if got.Location() != time.Local {
		t.Fatalf("expected local location")
	}
}

func TestParseReferenceDate_InvalidFormat(t *testing.T) {
	_, err := ParseReferenceDate("15-07-2026")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsFutureDate(t *testing.T) {
	today := TodayDate()
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	if IsFutureDate(yesterday) {
		t.Fatal("yesterday should not be future")
	}
	if !IsFutureDate(tomorrow) {
		t.Fatal("tomorrow should be future")
	}
	if IsFutureDate(today) {
		t.Fatal("today should not be future")
	}
}

func TestPreviousMonthRange(t *testing.T) {
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	start, end := PreviousMonthRange(ref)

	if start.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("start got %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-06-30" {
		t.Fatalf("end got %s", end.Format("2006-01-02"))
	}
}

func TestPreviousMonthRange_JanuaryReference(t *testing.T) {
	ref := time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)
	start, end := PreviousMonthRange(ref)

	if start.Format("2006-01-02") != "2025-12-01" {
		t.Fatalf("start got %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2025-12-31" {
		t.Fatalf("end got %s", end.Format("2006-01-02"))
	}
}

func TestSupportsPastDateRun(t *testing.T) {
	if !SupportsPastDateRun("leave_balance_job") {
		t.Fatal("leave_balance_job should support past date")
	}
	if !SupportsPastDateRun("work_day_report_job") {
		t.Fatal("work_day_report_job should support past date")
	}
	if SupportsPastDateRun("document_cleanup_job") {
		t.Fatal("document_cleanup_job should not support past date")
	}
	if SupportsPastDateRun("contract_status_info_job") {
		t.Fatal("contract_status_info_job should not support past date")
	}
}
