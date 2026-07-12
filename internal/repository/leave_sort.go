package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

var leaveRequestSortAllowlist = map[string]bool{
	"created_at":      true,
	"employee_name":   true,
	"leave_type_name": true,
	"start_date":      true,
	"end_date":        true,
	"requested_days":  true,
	// Legacy keys
	"employee_id":   true,
	"leave_type_id": true,
}

const leaveRequestDefaultSort = "created_at"

func normalizeLeaveRequestSortKey(sortKey string) string {
	switch sortKey {
	case "employee_id":
		return "employee_name"
	case "leave_type_id":
		return "leave_type_name"
	case "leave_type":
		return "leave_type_name"
	default:
		return types.AllowedSortOrDefault(sortKey, leaveRequestSortAllowlist, leaveRequestDefaultSort)
	}
}

// buildLeaveRequestOrderClause maps allowlisted keys to safe ORDER BY SQL.
func buildLeaveRequestOrderClause(sortKey, direction string) (orderClause string, needsEmployeeJoin bool, needsTypeJoin bool) {
	direction = types.NormalizeSortDirection(direction, string(types.DESC))
	key := normalizeLeaveRequestSortKey(sortKey)

	lr := domain.GetTableName("hr_leave_requests")
	emp := domain.GetTableName("hr_employees")
	lt := domain.GetTableName("hr_leave_types")
	nulls := "NULLS LAST"

	switch key {
	case "employee_name":
		return fmt.Sprintf("%s.first_name %s %s, %s.last_name %s %s", emp, direction, nulls, emp, direction, nulls), true, false
	case "leave_type_name":
		return fmt.Sprintf("%s.name %s %s", lt, direction, nulls), false, true
	case "start_date":
		return fmt.Sprintf("%s.start_date %s %s", lr, direction, nulls), false, false
	case "end_date":
		return fmt.Sprintf("%s.end_date %s %s", lr, direction, nulls), false, false
	case "requested_days":
		return fmt.Sprintf("%s.requested_days %s %s", lr, direction, nulls), false, false
	case "created_at":
		fallthrough
	default:
		return fmt.Sprintf("%s.created_at %s %s", lr, direction, nulls), false, false
	}
}
