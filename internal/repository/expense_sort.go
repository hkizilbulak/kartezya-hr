package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

var expenseRequestSortAllowlist = map[string]bool{
	"employee_name":     true,
	"expense_type_name": true,
	"description":       true,
	"amount":            true,
	"expense_date":      true,
	"created_at":        true,
}

const expenseRequestDefaultSort = "created_at"

func normalizeExpenseRequestSortKey(sortKey string) string {
	switch sortKey {
	case "employee_id":
		return "employee_name"
	case "expense_type_id":
		return "expense_type_name"
	default:
		return types.AllowedSortOrDefault(sortKey, expenseRequestSortAllowlist, expenseRequestDefaultSort)
	}
}

// buildExpenseRequestOrderClause maps allowlisted keys to safe ORDER BY SQL.
// employee_name / expense_type_name sort by displayed joined text, not FK ids.
func buildExpenseRequestOrderClause(sortKey, direction string) (orderClause string, needsEmployeeJoin bool, needsTypeJoin bool) {
	direction = types.NormalizeSortDirection(direction, string(types.DESC))
	key := normalizeExpenseRequestSortKey(sortKey)

	exp := domain.GetTableName("hr_expense_requests")
	emp := domain.GetTableName("hr_employees")
	etype := domain.GetTableName("hr_expense_types")
	nulls := "NULLS LAST"

	switch key {
	case "employee_name":
		return fmt.Sprintf("%s.first_name %s %s, %s.last_name %s %s", emp, direction, nulls, emp, direction, nulls), true, false
	case "expense_type_name":
		return fmt.Sprintf("%s.name %s %s", etype, direction, nulls), false, true
	case "description":
		return fmt.Sprintf("%s.description %s %s", exp, direction, nulls), false, false
	case "amount":
		return fmt.Sprintf("%s.amount %s %s", exp, direction, nulls), false, false
	case "expense_date":
		return fmt.Sprintf("%s.expense_date %s %s", exp, direction, nulls), false, false
	case "created_at":
		fallthrough
	default:
		return fmt.Sprintf("%s.created_at %s %s", exp, direction, nulls), false, false
	}
}
