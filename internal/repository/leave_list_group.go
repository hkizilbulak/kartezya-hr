package repository

import (
	"fmt"
	"time"
)

// buildLeaveListGroupClause returns the SQL predicate for admin leave pending/completed tabs.
// Count and data queries must both use this so filters stay identical.
func buildLeaveListGroupClause(listGroup, lrTable string) (clause string, args []any, ok bool) {
	switch listGroup {
	case "pending":
		return fmt.Sprintf(
			"(%s.status = ? OR (%s.status = ? AND %s.start_date::date > CURRENT_DATE))",
			lrTable, lrTable, lrTable,
		), []any{"PENDING", "APPROVED"}, true
	case "completed":
		// NULL start_date is treated as non-cancellable (completed), matching prior FE behavior.
		return fmt.Sprintf(
			"(%s.status IN (?, ?) OR (%s.status = ? AND (%s.start_date IS NULL OR %s.start_date::date <= CURRENT_DATE)))",
			lrTable, lrTable, lrTable, lrTable,
		), []any{"REJECTED", "CANCELLED", "APPROVED"}, true
	default:
		return "", nil, false
	}
}

// matchesLeaveListGroup mirrors buildLeaveListGroupClause for unit tests (date-truncated compare).
func matchesLeaveListGroup(listGroup, status string, startDate *time.Time, today time.Time) bool {
	today = truncateToDate(today)
	switch listGroup {
	case "pending":
		if status == "PENDING" {
			return true
		}
		return status == "APPROVED" && startDate != nil && truncateToDate(*startDate).After(today)
	case "completed":
		if status == "REJECTED" || status == "CANCELLED" {
			return true
		}
		if status != "APPROVED" {
			return false
		}
		if startDate == nil {
			return true
		}
		return !truncateToDate(*startDate).After(today) // <= today
	default:
		return false
	}
}

func truncateToDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
