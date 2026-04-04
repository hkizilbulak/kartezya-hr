package jobs

import (
	"log"

	"kartezya-hr/internal/service"
)

// DocumentCleanupJob handles cleanup of temporary/orphaned documents
type DocumentCleanupJob struct {
	documentService service.DocumentService
	hoursOld        int // Delete temporary files older than this many hours
}

// NewDocumentCleanupJob creates a new document cleanup job
func NewDocumentCleanupJob(documentService service.DocumentService, hoursOld int) *DocumentCleanupJob {
	if hoursOld <= 0 {
		hoursOld = 24 // Default: clean files older than 24 hours
	}

	return &DocumentCleanupJob{
		documentService: documentService,
		hoursOld:        hoursOld,
	}
}

// Run executes the cleanup job
func (j *DocumentCleanupJob) Run() {
	log.Printf("[DocumentCleanupJob] Starting cleanup of temporary files older than %d hours...", j.hoursOld)

	count, err := j.documentService.CleanupTemporaryFiles(j.hoursOld)
	if err != nil {
		log.Printf("[DocumentCleanupJob] Error during cleanup: %v", err)
		return
	}

	log.Printf("[DocumentCleanupJob] Cleanup completed. %d files removed.", count)
}
