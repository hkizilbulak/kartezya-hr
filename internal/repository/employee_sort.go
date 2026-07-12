package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

// employeeListSortAllowlist is the explicit FE→BE sort contract for employee lists.
var employeeListSortAllowlist = map[string]bool{
	"employee_name":   true,
	"company_name":    true,
	"department_name": true,
	"manager_name":    true,
	"hire_date":       true,
	// Legacy keys accepted and normalized to display-oriented keys.
	"first_name": true,
	"manager":    true,
}

const employeeListDefaultSort = "employee_name"

func normalizeEmployeeListSortKey(sortKey string) string {
	switch sortKey {
	case "first_name":
		return "employee_name"
	case "manager":
		return "manager_name"
	default:
		return types.AllowedSortOrDefault(sortKey, employeeListSortAllowlist, employeeListDefaultSort)
	}
}

// buildEmployeeListOrderClause maps an allowlisted sort key to a safe ORDER BY clause.
// Joined display fields use the latest work-information row (start_date DESC), matching list display.
func buildEmployeeListOrderClause(sortKey, direction string) string {
	direction = types.NormalizeSortDirection(direction, string(types.ASC))
	key := normalizeEmployeeListSortKey(sortKey)

	emp := domain.GetTableName("hr_employees")
	wi := domain.GetTableName("hr_employee_work_information")
	dept := domain.GetTableName("hr_departments")
	comp := domain.GetTableName("hr_companies")
	nulls := "NULLS LAST"

	switch key {
	case "company_name":
		return fmt.Sprintf(`(SELECT c.name FROM %s wi
			JOIN %s c ON c.id = wi.company_id AND c.deleted = false
			WHERE wi.employee_id = %s.id AND wi.deleted = false
			ORDER BY wi.start_date DESC LIMIT 1) %s %s`, wi, comp, emp, direction, nulls)
	case "department_name":
		return fmt.Sprintf(`(SELECT d.name FROM %s wi
			JOIN %s d ON d.id = wi.department_id AND d.deleted = false
			WHERE wi.employee_id = %s.id AND wi.deleted = false
			ORDER BY wi.start_date DESC LIMIT 1) %s %s`, wi, dept, emp, direction, nulls)
	case "manager_name":
		return fmt.Sprintf(`(SELECT d.manager FROM %s wi
			JOIN %s d ON d.id = wi.department_id AND d.deleted = false
			WHERE wi.employee_id = %s.id AND wi.deleted = false
			ORDER BY wi.start_date DESC LIMIT 1) %s %s`, wi, dept, emp, direction, nulls)
	case "hire_date":
		return fmt.Sprintf("%s.hire_date %s %s", emp, direction, nulls)
	case "employee_name":
		fallthrough
	default:
		return fmt.Sprintf("%s.first_name %s %s, %s.last_name %s %s", emp, direction, nulls, emp, direction, nulls)
	}
}
