package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

var (
	ErrJobAlreadySucceededForReferenceDate = errors.New("Bu görev seçilen tarih için daha önce başarıyla çalıştırılmış.")
	ErrJobAlreadyRunningForReferenceDate   = errors.New("Bu görev seçilen tarih için halen çalışıyor.")
	ErrJobReferenceDateConflict            = errors.New("Bu görev seçilen tarih için daha önce başarıyla çalıştırılmış veya halen çalışıyor.")
	ErrPastDateRunNotSupported             = errors.New("Bu görev geçmiş tarih için çalıştırılamaz.")
)

type JobService interface {
	SeedJobs() error
	GetAllJobs(sortParams types.SortParams) ([]domain.Job, error)
	GetActiveJobs() ([]domain.Job, error)
	GetJobByID(id uint) (*domain.Job, error)
	GetJobByKey(key string) (*domain.Job, error)
	UpdateJob(id uint, job *domain.Job, userID uint) error
	GetHistory(jobID uint, limit int) ([]domain.JobHistory, error)
	LogJobStart(jobID uint, referenceDate *time.Time, executionType string, triggeredByUserID *uint) (*domain.JobHistory, error)
	LogJobEnd(history *domain.JobHistory, status string, processedCount int, errSummary string) error
	ValidateReferenceDateRun(jobID uint, jobKey string, referenceDate time.Time) error
}

type jobService struct {
	jobRepo      repository.JobRepository
	auditService AuditService
}

func NewJobService(jobRepo repository.JobRepository, auditService AuditService) JobService {
	return &jobService{
		jobRepo:      jobRepo,
		auditService: auditService,
	}
}

// SeedJobs ensures that the default jobs exist in the database
func (s *jobService) SeedJobs() error {
	defaultJobs := []domain.Job{
		{
			JobKey:         "leave_balance_job",
			Name:           "Annual Leave Balance Update",
			CronExpression: "0 0 6 * * *", // At 06:00:00 every day
			IsActive:       true,
			TimeoutSecond:  3600,
		},
		{
			JobKey:         "document_cleanup_job",
			Name:           "Document Cleanup Job",
			CronExpression: "0 0 3 * * *", // At 03:00:00 every day
			IsActive:       true,
			TimeoutSecond:  3600,
		},
		{
			JobKey:         "contract_status_info_job",
			Name:           "Contract Status Report",
			CronExpression: "0 0 14 * * 1", // Every Monday at 14:00:00
			IsActive:       true,
			TimeoutSecond:  3600,
		},
		{
			JobKey:         "work_day_report_job",
			Name:           "Monthly Work Day Report",
			CronExpression: "0 0 0 10 * *", // 10th of every month at 00:00:00
			IsActive:       true,
			TimeoutSecond:  3600,
		},
	}

	for _, defaultJob := range defaultJobs {
		existing, err := s.jobRepo.GetByKey(defaultJob.JobKey)
		if err != nil {
			return fmt.Errorf("failed to check job existence %s: %w", defaultJob.JobKey, err)
		}

		if existing == nil {
			// Create it
			if err := s.jobRepo.Create(&defaultJob); err != nil {
				return fmt.Errorf("failed to create default job %s: %w", defaultJob.JobKey, err)
			}
		}
	}
	return nil
}

func (s *jobService) GetAllJobs(sortParams types.SortParams) ([]domain.Job, error) {
	return s.jobRepo.GetAll(sortParams)
}

func (s *jobService) GetActiveJobs() ([]domain.Job, error) {
	return s.jobRepo.GetActiveJobs()
}

func (s *jobService) GetJobByID(id uint) (*domain.Job, error) {
	return s.jobRepo.GetByID(id)
}

func (s *jobService) GetJobByKey(key string) (*domain.Job, error) {
	return s.jobRepo.GetByKey(key)
}

func (s *jobService) UpdateJob(id uint, updateData *domain.Job, userID uint) error {
	existing, err := s.jobRepo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("job not found")
	}

	// Update fields
	existing.CronExpression = updateData.CronExpression
	existing.IsActive = updateData.IsActive
	existing.TimeoutSecond = updateData.TimeoutSecond

	if err := s.jobRepo.Update(existing); err != nil {
		return err
	}

	// Audit log could go here
	modifiedBy := fmt.Sprintf("%d", userID)
	if s.auditService != nil {
		_ = s.auditService.CreateAuditLog("Job", existing.ID, domain.AuditActionUpdate, nil, existing, modifiedBy)
	}

	return nil
}

func (s *jobService) GetHistory(jobID uint, limit int) ([]domain.JobHistory, error) {
	return s.jobRepo.GetHistoryByJobID(jobID, limit)
}

func (s *jobService) LogJobStart(jobID uint, referenceDate *time.Time, executionType string, triggeredByUserID *uint) (*domain.JobHistory, error) {
	history := &domain.JobHistory{
		JobID:             jobID,
		StartTime:         time.Now(),
		Status:            "RUNNING",
		ExecutionType:     executionType,
		TriggeredByUserID: triggeredByUserID,
	}

	if executionType == "" {
		history.ExecutionType = "scheduled"
	}

	// Only persist reference_date when provided. Jobs without duplicate-date
	// semantics (e.g. cleanup) keep NULL so the partial unique index does not
	// block repeated same-day normal runs.
	if referenceDate != nil {
		refDate := time.Date(referenceDate.Year(), referenceDate.Month(), referenceDate.Day(), 0, 0, 0, 0, time.UTC)
		history.ReferenceDate = &refDate
	}

	if err := s.jobRepo.CreateHistory(history); err != nil {
		if repository.IsUniqueViolation(err) {
			return nil, ErrJobReferenceDateConflict
		}
		return nil, err
	}
	return history, nil
}

func (s *jobService) LogJobEnd(history *domain.JobHistory, status string, processedCount int, errSummary string) error {
	now := time.Now()
	history.EndTime = &now
	history.Status = status
	history.ProcessedCount = processedCount

	// Truncate error summary if it's too long
	if len(errSummary) > 500 {
		errSummary = errSummary[:500] + "..."
	}
	history.ErrorSummary = errSummary

	return s.jobRepo.UpdateHistory(history)
}

func (s *jobService) ValidateReferenceDateRun(jobID uint, jobKey string, referenceDate time.Time) error {
	if jobKey != "leave_balance_job" && jobKey != "work_day_report_job" {
		return nil
	}

	hasRunning, err := s.jobRepo.HasHistoryForReferenceDate(jobID, referenceDate, []string{"RUNNING"})
	if err != nil {
		return err
	}
	if hasRunning {
		return ErrJobAlreadyRunningForReferenceDate
	}

	hasSuccess, err := s.jobRepo.HasHistoryForReferenceDate(jobID, referenceDate, []string{"SUCCESS"})
	if err != nil {
		return err
	}
	if hasSuccess {
		return ErrJobAlreadySucceededForReferenceDate
	}

	return nil
}
