package jobs

import (
	"log"

	"kartezya-hr/internal/service"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler manages all scheduled jobs
type Scheduler struct {
	cron               *cron.Cron
	leaveBalanceJob    *LeaveBalanceJob
	documentCleanupJob *DocumentCleanupJob
}

// NewScheduler creates a new job scheduler
func NewScheduler(db *gorm.DB, documentService service.DocumentService) *Scheduler {
	// Create cron with seconds precision
	c := cron.New(cron.WithSeconds())

	return &Scheduler{
		cron:               c,
		leaveBalanceJob:    NewLeaveBalanceJob(db),
		documentCleanupJob: NewDocumentCleanupJob(documentService, 24), // Clean files older than 24 hours
	}
}

// Start begins running all scheduled jobs
func (s *Scheduler) Start() {
	log.Println("[Scheduler] Starting scheduled jobs...")

	// Schedule leave balance update job - every day at 06:00:00
	// Cron format: "seconds minutes hours day month weekday"
	// "0 0 6 * * *" = at 06:00:00 every day
	_, err := s.cron.AddFunc("0 0 6 * * *", s.leaveBalanceJob.Run)
	if err != nil {
		log.Printf("[Scheduler] Failed to schedule leave balance job: %v", err)
	} else {
		log.Println("[Scheduler] Leave balance update job scheduled to run daily at 06:00")
	}

	// Schedule document cleanup job - every day at 03:00:00
	// "0 0 3 * * *" = at 03:00:00 every day
	_, err = s.cron.AddFunc("0 0 3 * * *", s.documentCleanupJob.Run)
	if err != nil {
		log.Printf("[Scheduler] Failed to schedule document cleanup job: %v", err)
	} else {
		log.Println("[Scheduler] Document cleanup job scheduled to run daily at 03:00")
	}

	// Start the cron scheduler
	s.cron.Start()
	log.Println("[Scheduler] All jobs started successfully")
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("[Scheduler] Stopping scheduled jobs...")
	s.cron.Stop()
	log.Println("[Scheduler] All jobs stopped")
}

// RunLeaveBalanceJobNow runs the leave balance job immediately (for testing)
func (s *Scheduler) RunLeaveBalanceJobNow() {
	log.Println("[Scheduler] Manually triggering leave balance job...")
	s.leaveBalanceJob.Run()
}
