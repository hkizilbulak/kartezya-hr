package jobs

import (
	"fmt"
	"time"
)

const (
	ExecutionTypeScheduled      = "scheduled"
	ExecutionTypeManual         = "manual"
	ExecutionTypeManualBackfill = "manual_backfill"
)

// JobExecutionContext carries runtime metadata for a job execution.
type JobExecutionContext struct {
	ReferenceDate     time.Time
	ExecutionType     string
	TriggeredByUserID *uint
}

// JobFunc is the signature for all scheduled and manual job runners.
type JobFunc func(ctx JobExecutionContext) (int, error)

// TodayDate returns today's date at local midnight (date-only).
func TodayDate() time.Time {
	now := time.Now().In(time.Local)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
}

// ParseReferenceDate parses YYYY-MM-DD in the local timezone and normalizes to midnight.
func ParseReferenceDate(dateStr string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid reference_date format, use YYYY-MM-DD")
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
}

// IsFutureDate reports whether d is after today (date-only comparison).
func IsFutureDate(d time.Time) bool {
	return d.After(TodayDate())
}

// PreviousMonthRange returns the first and last day of the month before ref's month.
func PreviousMonthRange(ref time.Time) (time.Time, time.Time) {
	firstOfMonth := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, ref.Location())
	startDate := firstOfMonth.AddDate(0, -1, 0)
	endDate := firstOfMonth.AddDate(0, 0, -1)
	return startDate, endDate
}
