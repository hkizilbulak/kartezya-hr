package jobs

// PastDateSupportedJobKeys lists jobs that accept a manual backfill reference_date.
var PastDateSupportedJobKeys = map[string]bool{
	"leave_balance_job":   true,
	"work_day_report_job": true,
}

// DuplicateReferenceDateCheckJobKeys lists jobs guarded against duplicate reference_date runs.
var DuplicateReferenceDateCheckJobKeys = map[string]bool{
	"leave_balance_job":   true,
	"work_day_report_job": true,
}

func SupportsPastDateRun(jobKey string) bool {
	return PastDateSupportedJobKeys[jobKey]
}

func RequiresDuplicateReferenceDateCheck(jobKey string) bool {
	return DuplicateReferenceDateCheckJobKeys[jobKey]
}
