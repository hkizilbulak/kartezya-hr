package service

import (
	"errors"
	"testing"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"github.com/jackc/pgx/v5/pgconn"
)

type stubJobRepoForRun struct {
	hasRunning  bool
	hasSuccess  bool
	histories   []*domain.JobHistory
	createErr   error
	createCalls int
}

func (s *stubJobRepoForRun) Create(job *domain.Job) error { return nil }
func (s *stubJobRepoForRun) Update(job *domain.Job) error { return nil }
func (s *stubJobRepoForRun) GetByID(id uint) (*domain.Job, error) {
	return nil, nil
}
func (s *stubJobRepoForRun) GetByKey(key string) (*domain.Job, error) {
	return nil, nil
}
func (s *stubJobRepoForRun) GetAll(sortParams types.SortParams) ([]domain.Job, error) {
	return nil, nil
}
func (s *stubJobRepoForRun) GetActiveJobs() ([]domain.Job, error) { return nil, nil }
func (s *stubJobRepoForRun) CreateHistory(history *domain.JobHistory) error {
	s.createCalls++
	if s.createErr != nil {
		return s.createErr
	}
	s.histories = append(s.histories, history)
	return nil
}
func (s *stubJobRepoForRun) UpdateHistory(history *domain.JobHistory) error { return nil }
func (s *stubJobRepoForRun) GetHistoryByJobID(jobID uint, limit, offset int, sortParams types.SortParams) ([]domain.JobHistory, int64, error) {
	return nil, 0, nil
}
func (s *stubJobRepoForRun) HasHistoryForReferenceDate(jobID uint, referenceDate time.Time, statuses []string) (bool, error) {
	for _, st := range statuses {
		if st == "RUNNING" && s.hasRunning {
			return true, nil
		}
		if st == "SUCCESS" && s.hasSuccess {
			return true, nil
		}
	}
	return false, nil
}

func TestValidateReferenceDateRun_SkipsUnsupportedJob(t *testing.T) {
	svc := &jobService{jobRepo: &stubJobRepoForRun{hasSuccess: true}}
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	if err := svc.ValidateReferenceDateRun(1, "document_cleanup_job", ref); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateReferenceDateRun_RejectsSuccess(t *testing.T) {
	svc := &jobService{jobRepo: &stubJobRepoForRun{hasSuccess: true}}
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	err := svc.ValidateReferenceDateRun(1, "leave_balance_job", ref)
	if !errors.Is(err, ErrJobAlreadySucceededForReferenceDate) {
		t.Fatalf("expected success duplicate error, got %v", err)
	}
}

func TestValidateReferenceDateRun_RejectsRunning(t *testing.T) {
	svc := &jobService{jobRepo: &stubJobRepoForRun{hasRunning: true}}
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	err := svc.ValidateReferenceDateRun(2, "work_day_report_job", ref)
	if !errors.Is(err, ErrJobAlreadyRunningForReferenceDate) {
		t.Fatalf("expected running duplicate error, got %v", err)
	}
}

func TestValidateReferenceDateRun_AllowsWhenNoConflict(t *testing.T) {
	svc := &jobService{jobRepo: &stubJobRepoForRun{}}
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	if err := svc.ValidateReferenceDateRun(1, "leave_balance_job", ref); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateReferenceDateRun_FailedHistoryAllowsRetry(t *testing.T) {
	// Stub returns false for RUNNING/SUCCESS; FAILED rows are outside the unique index.
	svc := &jobService{jobRepo: &stubJobRepoForRun{}}
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	if err := svc.ValidateReferenceDateRun(1, "leave_balance_job", ref); err != nil {
		t.Fatalf("FAILED history must allow retry, got %v", err)
	}
}

func TestLogJobStart_PersistsAuditFields(t *testing.T) {
	repo := &stubJobRepoForRun{}
	svc := &jobService{jobRepo: repo}
	userID := uint(99)
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)

	history, err := svc.LogJobStart(5, &ref, "manual_backfill", &userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if history.ExecutionType != "manual_backfill" {
		t.Fatalf("got execution type %s", history.ExecutionType)
	}
	if history.TriggeredByUserID == nil || *history.TriggeredByUserID != userID {
		t.Fatal("expected triggered by user id")
	}
	if history.ReferenceDate == nil || history.ReferenceDate.Format("2006-01-02") != "2026-07-15" {
		t.Fatal("expected reference date persisted")
	}
	if len(repo.histories) != 1 {
		t.Fatalf("expected 1 history created, got %d", len(repo.histories))
	}
}

func TestLogJobStart_NullReferenceDate(t *testing.T) {
	repo := &stubJobRepoForRun{}
	svc := &jobService{jobRepo: repo}

	history, err := svc.LogJobStart(5, nil, "manual", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if history.ReferenceDate != nil {
		t.Fatal("expected nil reference_date for jobs outside duplicate protection")
	}
}

func TestLogJobStart_DefaultExecutionType(t *testing.T) {
	repo := &stubJobRepoForRun{}
	svc := &jobService{jobRepo: repo}
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)

	history, err := svc.LogJobStart(5, &ref, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if history.ExecutionType != "scheduled" {
		t.Fatalf("expected scheduled default, got %s", history.ExecutionType)
	}
}

func TestLogJobStart_MapsUniqueViolationToConflict(t *testing.T) {
	repo := &stubJobRepoForRun{
		createErr: &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"},
	}
	svc := &jobService{jobRepo: repo}
	ref := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)

	history, err := svc.LogJobStart(5, &ref, "manual_backfill", nil)
	if history != nil {
		t.Fatal("expected nil history on unique violation")
	}
	if !errors.Is(err, ErrJobReferenceDateConflict) {
		t.Fatalf("expected ErrJobReferenceDateConflict, got %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected create attempted once, got %d", repo.createCalls)
	}
}
