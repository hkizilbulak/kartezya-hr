package jobs

import (
	"testing"
	"time"
)

func TestHistoryReferenceDateForJob_ProtectedJobsPersistDate(t *testing.T) {
	ctx := JobExecutionContext{
		ReferenceDate: time.Date(2026, 7, 15, 12, 30, 0, 0, time.Local),
		ExecutionType: ExecutionTypeManual,
	}

	for _, key := range []string{"leave_balance_job", "work_day_report_job"} {
		got := historyReferenceDateForJob(key, ctx)
		if got == nil {
			t.Fatalf("%s: expected reference date", key)
		}
		if got.Format("2006-01-02") != "2026-07-15" {
			t.Fatalf("%s: got %s", key, got.Format("2006-01-02"))
		}
	}
}

func TestHistoryReferenceDateForJob_UnprotectedJobsKeepNull(t *testing.T) {
	ctx := JobExecutionContext{
		ReferenceDate: TodayDate(),
		ExecutionType: ExecutionTypeManual,
	}

	for _, key := range []string{"document_cleanup_job", "contract_status_info_job"} {
		got := historyReferenceDateForJob(key, ctx)
		if got != nil {
			t.Fatalf("%s: expected nil reference_date for normal run, got %v", key, got)
		}
	}
}

func TestHistoryReferenceDateForJob_BackfillAlwaysPersists(t *testing.T) {
	ctx := JobExecutionContext{
		ReferenceDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
		ExecutionType: ExecutionTypeManualBackfill,
	}
	got := historyReferenceDateForJob("leave_balance_job", ctx)
	if got == nil || got.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("expected backfill reference date, got %v", got)
	}
}
