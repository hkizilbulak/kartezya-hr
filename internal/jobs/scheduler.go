package jobs

import (
	"log"
	"os"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler manages all scheduled jobs
type Scheduler struct {
	cron               *cron.Cron
	jobService         service.JobService
	leaveBalanceJob    *LeaveBalanceJob
	documentCleanupJob *DocumentCleanupJob
	contractInfoJob    *ContractStatusInfoJob
	workDayReportJob   *WorkDayReportJob
	registry           map[string]JobFunc
	entryIDs           map[string]cron.EntryID
}

// NewScheduler creates a new job scheduler
func NewScheduler(db *gorm.DB, documentService service.DocumentService, jobService service.JobService,
	emailService service.EmailService, mailConfigService service.MailConfigService,
	reportService service.ReportService) *Scheduler {
	c := cron.New(cron.WithSeconds())

	leaveBalanceJob := NewLeaveBalanceJob(db)
	documentCleanupJob := NewDocumentCleanupJob(documentService, 24)
	contractInfoJob := NewContractStatusInfoJob(db, emailService, mailConfigService)
	workDayReportJob := NewWorkDayReportJob(reportService, emailService, mailConfigService)

	registry := map[string]JobFunc{
		"leave_balance_job":        leaveBalanceJob.Run,
		"document_cleanup_job":     documentCleanupJob.Run,
		"contract_status_info_job": contractInfoJob.Run,
		"work_day_report_job":      workDayReportJob.Run,
	}

	return &Scheduler{
		cron:               c,
		jobService:         jobService,
		leaveBalanceJob:    leaveBalanceJob,
		documentCleanupJob: documentCleanupJob,
		contractInfoJob:    contractInfoJob,
		workDayReportJob:   workDayReportJob,
		registry:           registry,
		entryIDs:           make(map[string]cron.EntryID),
	}
}

// Start begins running all active scheduled jobs
func (s *Scheduler) Start() {
	log.Println("[Scheduler] Starting scheduled jobs...")

	if err := s.jobService.SeedJobs(); err != nil {
		log.Printf("[Scheduler] Error seeding jobs: %v", err)
	}

	activeJobs, err := s.jobService.GetActiveJobs()
	if err != nil {
		log.Printf("[Scheduler] Failed to get active jobs: %v", err)
		return
	}

	for _, job := range activeJobs {
		s.scheduleJob(job)
	}

	s.cron.Start()
	log.Println("[Scheduler] All active jobs started successfully")
}

func (s *Scheduler) scheduleJob(job domain.Job) {
	jobFunc, exists := s.registry[job.JobKey]
	if !exists {
		log.Printf("[Scheduler] Job function for key %s not found in registry", job.JobKey)
		return
	}

	if entryID, ok := s.entryIDs[job.JobKey]; ok {
		s.cron.Remove(entryID)
	}

	wrapper := func() {
		ctx := JobExecutionContext{
			ReferenceDate: TodayDate(),
			ExecutionType: ExecutionTypeScheduled,
		}
		s.executeJob(job.ID, job.JobKey, jobFunc, ctx)
	}

	entryID, err := s.cron.AddFunc(job.CronExpression, wrapper)
	if err != nil {
		log.Printf("[Scheduler] Failed to schedule job %s: %v", job.JobKey, err)
	} else {
		s.entryIDs[job.JobKey] = entryID
		log.Printf("[Scheduler] Scheduled job %s with cron %s", job.JobKey, job.CronExpression)
	}
}

// historyReferenceDateForJob returns a persisted reference_date only for jobs that
// participate in duplicate-date protection. Other jobs keep NULL so repeated
// same-day normal runs are not blocked by the partial unique index.
func historyReferenceDateForJob(jobKey string, ctx JobExecutionContext) *time.Time {
	if RequiresDuplicateReferenceDateCheck(jobKey) || ctx.ExecutionType == ExecutionTypeManualBackfill {
		ref := time.Date(ctx.ReferenceDate.Year(), ctx.ReferenceDate.Month(), ctx.ReferenceDate.Day(), 0, 0, 0, 0, time.Local)
		return &ref
	}
	return nil
}

func (s *Scheduler) executeJob(jobID uint, jobKey string, jobFunc JobFunc, ctx JobExecutionContext) {
	log.Printf("[Scheduler] Executing job: %s (reference_date=%s, type=%s)",
		jobKey, ctx.ReferenceDate.Format("2006-01-02"), ctx.ExecutionType)

	history, err := s.jobService.LogJobStart(jobID, historyReferenceDateForJob(jobKey, ctx), ctx.ExecutionType, ctx.TriggeredByUserID)
	if err != nil {
		log.Printf("[Scheduler] Failed to log job start for %s: %v", jobKey, err)
		return
	}

	s.runJobWithHistory(jobID, jobKey, jobFunc, ctx, history)
}

func (s *Scheduler) runJobWithHistory(jobID uint, jobKey string, jobFunc JobFunc, ctx JobExecutionContext, history *domain.JobHistory) {
	hostname, _ := os.Hostname()
	history.ExecutionNode = hostname

	processedCount, err := jobFunc(ctx)

	status := "SUCCESS"
	errSummary := ""
	if err != nil {
		status = "FAILED"
		errSummary = err.Error()
	}

	if updateErr := s.jobService.LogJobEnd(history, status, processedCount, errSummary); updateErr != nil {
		log.Printf("[Scheduler] Failed to log job end for %s: %v", jobKey, updateErr)
	}

	log.Printf("[Scheduler] Finished job: %s, Status: %s", jobKey, status)
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("[Scheduler] Stopping scheduled jobs...")
	s.cron.Stop()
	log.Println("[Scheduler] All jobs stopped")
}

// ReloadJob updates the cron schedule for a specific job
func (s *Scheduler) ReloadJob(jobKey string) {
	job, err := s.jobService.GetJobByKey(jobKey)
	if err != nil || job == nil {
		log.Printf("[Scheduler] Job %s not found for reload", jobKey)
		return
	}

	if entryID, ok := s.entryIDs[jobKey]; ok {
		s.cron.Remove(entryID)
		delete(s.entryIDs, jobKey)
		log.Printf("[Scheduler] Removed old schedule for job %s", jobKey)
	}

	if job.IsActive {
		s.scheduleJob(*job)
	}
}

// TriggerJob runs the specified job immediately with the given execution context.
// RUNNING history is inserted BEFORE the job goroutine starts. If insert fails
// (including unique violation), the job is not started.
func (s *Scheduler) TriggerJob(jobKey string, ctx JobExecutionContext) error {
	job, err := s.jobService.GetJobByKey(jobKey)
	if err != nil {
		return err
	}
	if job == nil {
		return log.Output(2, "[Scheduler] Job not found for trigger")
	}

	jobFunc, exists := s.registry[jobKey]
	if !exists {
		return log.Output(2, "[Scheduler] Job function not found for trigger")
	}

	log.Printf("[Scheduler] Manually triggering job: %s (reference_date=%s, type=%s)",
		jobKey, ctx.ReferenceDate.Format("2006-01-02"), ctx.ExecutionType)

	history, err := s.jobService.LogJobStart(job.ID, historyReferenceDateForJob(jobKey, ctx), ctx.ExecutionType, ctx.TriggeredByUserID)
	if err != nil {
		return err
	}

	go s.runJobWithHistory(job.ID, jobKey, jobFunc, ctx, history)

	return nil
}
