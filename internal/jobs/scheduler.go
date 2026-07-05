package jobs

import (
	"log"
	"os"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type JobFunc func() (int, error)

// Scheduler manages all scheduled jobs
type Scheduler struct {
	cron               *cron.Cron
	jobService         service.JobService
	leaveBalanceJob    *LeaveBalanceJob
	documentCleanupJob *DocumentCleanupJob
	contractInfoJob    *ContractStatusInfoJob
	registry           map[string]JobFunc
	entryIDs           map[string]cron.EntryID
}

// NewScheduler creates a new job scheduler
func NewScheduler(db *gorm.DB, documentService service.DocumentService, jobService service.JobService,
	emailService service.EmailService, mailConfigService service.MailConfigService) *Scheduler {
	c := cron.New(cron.WithSeconds())

	leaveBalanceJob := NewLeaveBalanceJob(db)
	documentCleanupJob := NewDocumentCleanupJob(documentService, 24)
	contractInfoJob := NewContractStatusInfoJob(db, emailService, mailConfigService)

	registry := map[string]JobFunc{
		"leave_balance_job":        leaveBalanceJob.Run,
		"document_cleanup_job":     documentCleanupJob.Run,
		"contract_status_info_job": contractInfoJob.Run,
	}

	return &Scheduler{
		cron:               c,
		jobService:         jobService,
		leaveBalanceJob:    leaveBalanceJob,
		documentCleanupJob: documentCleanupJob,
		contractInfoJob:    contractInfoJob,
		registry:           registry,
		entryIDs:           make(map[string]cron.EntryID),
	}
}

// Start begins running all active scheduled jobs
func (s *Scheduler) Start() {
	log.Println("[Scheduler] Starting scheduled jobs...")

	// Seed jobs if not present
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

	// Start the cron scheduler
	s.cron.Start()
	log.Println("[Scheduler] All active jobs started successfully")
}

func (s *Scheduler) scheduleJob(job domain.Job) {
	jobFunc, exists := s.registry[job.JobKey]
	if !exists {
		log.Printf("[Scheduler] Job function for key %s not found in registry", job.JobKey)
		return
	}

	// Remove existing if any
	if entryID, ok := s.entryIDs[job.JobKey]; ok {
		s.cron.Remove(entryID)
	}

	wrapper := func() {
		s.executeJob(job.ID, job.JobKey, jobFunc)
	}

	entryID, err := s.cron.AddFunc(job.CronExpression, wrapper)
	if err != nil {
		log.Printf("[Scheduler] Failed to schedule job %s: %v", job.JobKey, err)
	} else {
		s.entryIDs[job.JobKey] = entryID
		log.Printf("[Scheduler] Scheduled job %s with cron %s", job.JobKey, job.CronExpression)
	}
}

func (s *Scheduler) executeJob(jobID uint, jobKey string, jobFunc JobFunc) {
	log.Printf("[Scheduler] Executing job: %s", jobKey)

	history, err := s.jobService.LogJobStart(jobID)
	if err != nil {
		log.Printf("[Scheduler] Failed to log job start for %s: %v", jobKey, err)
		return
	}

	hostname, _ := os.Hostname()
	history.ExecutionNode = hostname

	processedCount, err := jobFunc()

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

// TriggerJobNow runs the specified job immediately (does not affect cron schedule)
func (s *Scheduler) TriggerJobNow(jobKey string) error {
	job, err := s.jobService.GetJobByKey(jobKey)
	if err != nil {
		return err
	}
	if job == nil {
		return log.Output(2, "[Scheduler] Job not found for trigger") // Not a real error but handled by handler anyway
	}

	jobFunc, exists := s.registry[jobKey]
	if !exists {
		return log.Output(2, "[Scheduler] Job function not found for trigger")
	}

	log.Printf("[Scheduler] Manually triggering job: %s", jobKey)

	// Run in background
	go s.executeJob(job.ID, jobKey, jobFunc)

	return nil
}
