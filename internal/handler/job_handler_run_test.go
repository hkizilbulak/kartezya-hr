package handler

import (
	"errors"
	"testing"

	"kartezya-hr/internal/jobs"
	"kartezya-hr/internal/service"
)

func TestBuildJobExecutionContext_NoBody(t *testing.T) {
	userID := uint(42)
	ctx, err := buildJobExecutionContext(RunJobRequest{}, userID, "leave_balance_job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ExecutionType != jobs.ExecutionTypeManual {
		t.Fatalf("got execution type %s", ctx.ExecutionType)
	}
	if ctx.TriggeredByUserID == nil || *ctx.TriggeredByUserID != userID {
		t.Fatal("expected triggered by user")
	}
	if ctx.ReferenceDate.Format("2006-01-02") != jobs.TodayDate().Format("2006-01-02") {
		t.Fatalf("expected today reference date, got %s", ctx.ReferenceDate.Format("2006-01-02"))
	}
}

func TestBuildJobExecutionContext_EmptyReferenceDate(t *testing.T) {
	userID := uint(7)
	ctx, err := buildJobExecutionContext(RunJobRequest{ReferenceDate: "   "}, userID, "work_day_report_job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ExecutionType != jobs.ExecutionTypeManual {
		t.Fatalf("got execution type %s", ctx.ExecutionType)
	}
}

func TestBuildJobExecutionContext_ValidPastDate(t *testing.T) {
	userID := uint(5)
	ctx, err := buildJobExecutionContext(RunJobRequest{ReferenceDate: "2026-07-15"}, userID, "leave_balance_job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ExecutionType != jobs.ExecutionTypeManualBackfill {
		t.Fatalf("got execution type %s", ctx.ExecutionType)
	}
	if ctx.ReferenceDate.Format("2006-01-02") != "2026-07-15" {
		t.Fatalf("got reference date %s", ctx.ReferenceDate.Format("2006-01-02"))
	}
}

func TestBuildJobExecutionContext_UnsupportedJob(t *testing.T) {
	_, err := buildJobExecutionContext(RunJobRequest{ReferenceDate: "2026-07-15"}, 1, "document_cleanup_job")
	if !errors.Is(err, service.ErrPastDateRunNotSupported) {
		t.Fatalf("expected ErrPastDateRunNotSupported, got %v", err)
	}
}

func TestBuildJobExecutionContext_ContractStatusUnsupported(t *testing.T) {
	_, err := buildJobExecutionContext(RunJobRequest{ReferenceDate: "2026-07-15"}, 1, "contract_status_info_job")
	if !errors.Is(err, service.ErrPastDateRunNotSupported) {
		t.Fatalf("expected ErrPastDateRunNotSupported, got %v", err)
	}
}

func TestBuildJobExecutionContext_FutureDate(t *testing.T) {
	future := jobs.TodayDate().AddDate(0, 0, 2).Format("2006-01-02")
	_, err := buildJobExecutionContext(RunJobRequest{ReferenceDate: future}, 1, "leave_balance_job")
	if err == nil {
		t.Fatal("expected future date error")
	}
}

func TestBuildJobExecutionContext_InvalidFormat(t *testing.T) {
	_, err := buildJobExecutionContext(RunJobRequest{ReferenceDate: "2026/07/15"}, 1, "leave_balance_job")
	if err == nil {
		t.Fatal("expected invalid format error")
	}
}

func TestBuildJobExecutionContext_TodayAsBackfill(t *testing.T) {
	today := jobs.TodayDate().Format("2006-01-02")
	ctx, err := buildJobExecutionContext(RunJobRequest{ReferenceDate: today}, 3, "work_day_report_job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ExecutionType != jobs.ExecutionTypeManualBackfill {
		t.Fatalf("expected manual_backfill, got %s", ctx.ExecutionType)
	}
}
