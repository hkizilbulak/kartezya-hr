package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

var otherRequestSortAllowlist = map[string]bool{
	"employee_name":     true,
	"request_type_name": true,
	"description":       true,
	"created_at":        true,
	"id":                true,
	// Legacy
	"employee_id":     true,
	"request_type_id": true,
}

const otherRequestDefaultSort = "created_at"

func normalizeOtherRequestSortKey(sortKey string) string {
	switch sortKey {
	case "employee_id":
		return "employee_name"
	case "request_type_id":
		return "request_type_name"
	case "request_type":
		return "request_type_name"
	default:
		return types.AllowedSortOrDefault(sortKey, otherRequestSortAllowlist, otherRequestDefaultSort)
	}
}

func buildOtherRequestOrderClause(sortKey, direction string) (orderClause string, needsEmployeeJoin bool, needsTypeJoin bool) {
	direction = types.NormalizeSortDirection(direction, string(types.ASC))
	key := normalizeOtherRequestSortKey(sortKey)

	req := domain.GetTableName("hr_other_requests")
	emp := domain.GetTableName("hr_employees")
	rt := domain.GetTableName("hr_request_types")
	nulls := "NULLS LAST"

	switch key {
	case "employee_name":
		return fmt.Sprintf("%s.first_name %s %s, %s.last_name %s %s", emp, direction, nulls, emp, direction, nulls), true, false
	case "request_type_name":
		return fmt.Sprintf("%s.name %s %s", rt, direction, nulls), false, true
	case "description":
		return fmt.Sprintf("%s.description %s %s", req, direction, nulls), false, false
	case "id":
		return fmt.Sprintf("%s.id %s %s", req, direction, nulls), false, false
	case "created_at":
		fallthrough
	default:
		return fmt.Sprintf("%s.created_at %s %s", req, direction, nulls), false, false
	}
}
