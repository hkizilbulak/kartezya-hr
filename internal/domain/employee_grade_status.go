package domain

import "time"

// EmployeeGradeActiveCandidate is a deleted=false history row used to pick
// the single ACTIVE grade deterministically (mirrors migration ORDER BY).
type EmployeeGradeActiveCandidate struct {
	ID        uint
	StartDate time.Time
	EndDate   *time.Time
	CreatedAt time.Time
	// Status is optional pre-migration; empty means unknown / not yet backfilled.
	Status EmployeeGradeStatus
}

// SelectActiveEmployeeGradeID picks the winning ACTIVE row for one employee.
// Priority (stable / deterministic):
//  1. status = ACTIVE (when Status is set)
//  2. end_date IS NULL
//  3. asOf inside [start_date, end_date] (open end counts as in range)
//  4. start_date DESC
//  5. created_at DESC
//  6. id DESC
//
// Returns 0 when candidates is empty.
func SelectActiveEmployeeGradeID(candidates []EmployeeGradeActiveCandidate, asOf time.Time) uint {
	if len(candidates) == 0 {
		return 0
	}

	asOfDay := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)

	bestIdx := 0
	for i := 1; i < len(candidates); i++ {
		if employeeGradeActiveCandidateLess(candidates[i], candidates[bestIdx], asOfDay) {
			bestIdx = i
		}
	}
	return candidates[bestIdx].ID
}

// employeeGradeActiveCandidateLess reports whether a should rank before b
// (i.e. a is a better ACTIVE candidate).
func employeeGradeActiveCandidateLess(a, b EmployeeGradeActiveCandidate, asOfDay time.Time) bool {
	aStatusActive := a.Status == EmployeeGradeStatusActive
	bStatusActive := b.Status == EmployeeGradeStatusActive
	if aStatusActive != bStatusActive {
		return aStatusActive
	}

	aOpen := a.EndDate == nil
	bOpen := b.EndDate == nil
	if aOpen != bOpen {
		return aOpen
	}

	aInRange := employeeGradeInRange(a.StartDate, a.EndDate, asOfDay)
	bInRange := employeeGradeInRange(b.StartDate, b.EndDate, asOfDay)
	if aInRange != bInRange {
		return aInRange
	}

	if !a.StartDate.Equal(b.StartDate) {
		return a.StartDate.After(b.StartDate)
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	return a.ID > b.ID
}

func employeeGradeInRange(start time.Time, end *time.Time, asOfDay time.Time) bool {
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	if startDay.After(asOfDay) {
		return false
	}
	if end == nil {
		return true
	}
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	return !endDay.Before(asOfDay)
}

// CloseEndDateForInactive returns a deterministic end_date when demoting a row
// to INACTIVE. Prefer nextStart-1 day when that stays >= startDate; otherwise
// startDate (same-day close). nextStart nil ⇒ startDate.
func CloseEndDateForInactive(startDate time.Time, nextStart *time.Time) time.Time {
	startDay := dateOnlyUTC(startDate)
	if nextStart == nil {
		return startDay
	}
	nextDay := dateOnlyUTC(*nextStart)
	candidate := nextDay.AddDate(0, 0, -1)
	if candidate.Before(startDay) {
		return startDay
	}
	return candidate
}

// ActiveGradeCloseEndDate computes end_date for closing an ACTIVE grade when a
// new assignment starts on newStart. Result is newStart-1 day (DATE semantics, UTC calendar).
// Returns ErrEmployeeGradeInvalidCloseDate when that end_date would be before activeStart
// (includes same-day reassignment).
func ActiveGradeCloseEndDate(activeStart, newStart time.Time) (time.Time, error) {
	activeDay := dateOnlyUTC(activeStart)
	newDay := dateOnlyUTC(newStart)
	endDay := newDay.AddDate(0, 0, -1)
	if endDay.Before(activeDay) {
		return time.Time{}, ErrEmployeeGradeInvalidCloseDate
	}
	return endDay, nil
}

func dateOnlyUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
