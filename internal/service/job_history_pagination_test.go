package service

import (
	"testing"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

type stubJobRepoForHistory struct {
	rows           []domain.JobHistory
	total          int64
	err            error
	lastLimit      int
	lastOffset     int
	lastSortParams types.SortParams
	lastJobID      uint
}

func (s *stubJobRepoForHistory) Create(job *domain.Job) error { return nil }
func (s *stubJobRepoForHistory) Update(job *domain.Job) error { return nil }
func (s *stubJobRepoForHistory) GetByID(id uint) (*domain.Job, error) {
	return nil, nil
}
func (s *stubJobRepoForHistory) GetByKey(key string) (*domain.Job, error) {
	return nil, nil
}
func (s *stubJobRepoForHistory) GetAll(sortParams types.SortParams) ([]domain.Job, error) {
	return nil, nil
}
func (s *stubJobRepoForHistory) GetActiveJobs() ([]domain.Job, error) { return nil, nil }
func (s *stubJobRepoForHistory) CreateHistory(history *domain.JobHistory) error {
	return nil
}
func (s *stubJobRepoForHistory) UpdateHistory(history *domain.JobHistory) error {
	return nil
}
func (s *stubJobRepoForHistory) GetHistoryByJobID(jobID uint, limit, offset int, sortParams types.SortParams) ([]domain.JobHistory, int64, error) {
	s.lastJobID = jobID
	s.lastLimit = limit
	s.lastOffset = offset
	s.lastSortParams = sortParams
	if s.err != nil {
		return nil, 0, s.err
	}
	return s.rows, s.total, nil
}
func (s *stubJobRepoForHistory) HasHistoryForReferenceDate(jobID uint, referenceDate time.Time, statuses []string) (bool, error) {
	return false, nil
}

func TestGetHistoryDefaultsAndPagination(t *testing.T) {
	repo := &stubJobRepoForHistory{
		rows:  []domain.JobHistory{{ID: 1, JobID: 7, Status: "SUCCESS"}},
		total: 42,
	}
	svc := NewJobService(repo, nil)

	result, err := svc.GetHistory(7, 0, 0, types.SortParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastJobID != 7 {
		t.Fatalf("jobID: got %d", repo.lastJobID)
	}
	if repo.lastLimit != 10 || repo.lastOffset != 0 {
		t.Fatalf("expected limit=10 offset=0, got limit=%d offset=%d", repo.lastLimit, repo.lastOffset)
	}
	if repo.lastSortParams.Sort != "start_time" || repo.lastSortParams.Direction != "DESC" {
		t.Fatalf("expected start_time DESC, got %s %s", repo.lastSortParams.Sort, repo.lastSortParams.Direction)
	}
	if result.Page.Page != 1 || result.Page.Limit != 10 {
		t.Fatalf("page meta: %+v", result.Page)
	}
	if result.Page.Total != 42 || result.Page.TotalPages != 5 {
		t.Fatalf("expected total=42 total_pages=5, got total=%d total_pages=%d", result.Page.Total, result.Page.TotalPages)
	}
	if result.Page.Sort != "start_time" || result.Page.Direction != "DESC" {
		t.Fatalf("sort meta: %+v", result.Page)
	}
}

func TestGetHistorySecondPageOffset(t *testing.T) {
	repo := &stubJobRepoForHistory{total: 25}
	svc := NewJobService(repo, nil)

	result, err := svc.GetHistory(1, 2, 10, types.SortParams{Sort: "start_time", Direction: "DESC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastOffset != 10 || repo.lastLimit != 10 {
		t.Fatalf("expected offset=10 limit=10, got offset=%d limit=%d", repo.lastOffset, repo.lastLimit)
	}
	if result.Page.Page != 2 || result.Page.TotalPages != 3 {
		t.Fatalf("page=%d total_pages=%d", result.Page.Page, result.Page.TotalPages)
	}
}

func TestGetHistoryCustomLimits(t *testing.T) {
	repo := &stubJobRepoForHistory{total: 100}
	svc := NewJobService(repo, nil)

	for _, limit := range []int{20, 50, 100} {
		_, err := svc.GetHistory(1, 1, limit, types.SortParams{Sort: "status", Direction: "ASC"})
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if repo.lastLimit != limit {
			t.Fatalf("expected limit=%d, got %d", limit, repo.lastLimit)
		}
	}
}

func TestGetHistoryInvalidPageAndLimit(t *testing.T) {
	repo := &stubJobRepoForHistory{total: 0}
	svc := NewJobService(repo, nil)

	result, err := svc.GetHistory(1, -5, -1, types.SortParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Page.Page != 1 || result.Page.Limit != 10 {
		t.Fatalf("expected page=1 limit=10, got page=%d limit=%d", result.Page.Page, result.Page.Limit)
	}

	_, err = svc.GetHistory(1, 1, 101, types.SortParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastLimit != 10 {
		t.Fatalf("limit>100 should clamp to 10, got %d", repo.lastLimit)
	}
}

func TestGetHistoryInvalidDirectionNormalized(t *testing.T) {
	repo := &stubJobRepoForHistory{}
	svc := NewJobService(repo, nil)

	result, err := svc.GetHistory(1, 1, 10, types.SortParams{Sort: "end_time", Direction: "sideways"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSortParams.Direction != "DESC" || result.Page.Direction != "DESC" {
		t.Fatalf("invalid direction should fall back to DESC, got %s", result.Page.Direction)
	}
	if repo.lastSortParams.Sort != "end_time" || result.Page.Sort != "end_time" {
		t.Fatalf("valid sort should be preserved, got repo=%s page=%s", repo.lastSortParams.Sort, result.Page.Sort)
	}
}

func TestGetHistoryInvalidSortNormalizedInMetadata(t *testing.T) {
	repo := &stubJobRepoForHistory{}
	svc := NewJobService(repo, nil)

	result, err := svc.GetHistory(1, 1, 10, types.SortParams{Sort: "invalid_field", Direction: "wrong"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSortParams.Sort != "start_time" || repo.lastSortParams.Direction != "DESC" {
		t.Fatalf("repo should receive start_time DESC, got %s %s", repo.lastSortParams.Sort, repo.lastSortParams.Direction)
	}
	if result.Page.Sort != "start_time" || result.Page.Direction != "DESC" {
		t.Fatalf("page metadata should match repo sort, got %s %s", result.Page.Sort, result.Page.Direction)
	}
}

func TestGetHistoryValidSortDirectionMatchMetadata(t *testing.T) {
	repo := &stubJobRepoForHistory{}
	svc := NewJobService(repo, nil)

	result, err := svc.GetHistory(1, 1, 10, types.SortParams{Sort: "status", Direction: "ASC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSortParams.Sort != "status" || repo.lastSortParams.Direction != "ASC" {
		t.Fatalf("repo got %s %s", repo.lastSortParams.Sort, repo.lastSortParams.Direction)
	}
	if result.Page.Sort != "status" || result.Page.Direction != "ASC" {
		t.Fatalf("page got %s %s", result.Page.Sort, result.Page.Direction)
	}
}

func TestGetHistoryEmptyResult(t *testing.T) {
	repo := &stubJobRepoForHistory{
		rows:  []domain.JobHistory{},
		total: 0,
	}
	svc := NewJobService(repo, nil)

	result, err := svc.GetHistory(1, 1, 10, types.SortParams{Sort: "processed_count", Direction: "ASC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, ok := result.Data.([]domain.JobHistory)
	if !ok {
		t.Fatalf("data type: %T", result.Data)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %d", len(data))
	}
	if result.Page.Total != 0 || result.Page.TotalPages != 0 {
		t.Fatalf("expected total=0 total_pages=0, got total=%d total_pages=%d", result.Page.Total, result.Page.TotalPages)
	}
}

func TestGetHistoryAllowedSortKeysPassedThrough(t *testing.T) {
	repo := &stubJobRepoForHistory{}
	svc := NewJobService(repo, nil)

	keys := []string{"start_time", "end_time", "processed_count", "status", "id", "reference_date"}
	for _, key := range keys {
		_, err := svc.GetHistory(1, 1, 10, types.SortParams{Sort: key, Direction: "ASC"})
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if repo.lastSortParams.Sort != key {
			t.Fatalf("expected sort=%s, got %s", key, repo.lastSortParams.Sort)
		}
	}
}
