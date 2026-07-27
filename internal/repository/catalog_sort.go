package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

var companySortAllowlist = map[string]bool{
	"id":         true,
	"name":       true,
	"address":    true,
	"phone":      true,
	"email":      true,
	"website":    true,
	"created_at": true,
	"updated_at": true,
}

const companyDefaultSort = "id"

func buildCompanyOrderClause(sortKey, direction string) string {
	return buildAllowlistedOrder(sortKey, direction, companyDefaultSort, string(types.ASC), companySortAllowlist)
}

var departmentSortAllowlist = map[string]bool{
	"company":      true,
	"company_name": true,
	"name":         true,
	"manager":      true,
	"created_at":   true,
	"updated_at":   true,
	"id":           true,
}

const departmentDefaultSort = "id"

func normalizeDepartmentSortKey(sortKey string) string {
	switch sortKey {
	case "company_name":
		return "company"
	default:
		return types.AllowedSortOrDefault(sortKey, departmentSortAllowlist, departmentDefaultSort)
	}
}

func buildDepartmentOrderClause(sortKey, direction string) string {
	direction = types.NormalizeSortDirection(direction, string(types.ASC))
	key := normalizeDepartmentSortKey(sortKey)
	dept := domain.GetTableName("hr_departments")
	company := domain.GetTableName("hr_companies")

	switch key {
	case "company":
		return fmt.Sprintf("%s.name %s", company, direction)
	case "name":
		return fmt.Sprintf("%s.name %s", dept, direction)
	case "manager":
		return fmt.Sprintf("%s.manager %s", dept, direction)
	case "created_at":
		return fmt.Sprintf("%s.created_at %s", dept, direction)
	case "updated_at":
		return fmt.Sprintf("%s.updated_at %s", dept, direction)
	case "id":
		fallthrough
	default:
		return fmt.Sprintf("%s.id %s", dept, direction)
	}
}

var jobPositionSortAllowlist = map[string]bool{
	"id":         true,
	"title":      true,
	"created_at": true,
	"updated_at": true,
}

const jobPositionDefaultSort = "id"

func buildJobPositionOrderClause(sortKey, direction string) string {
	return buildAllowlistedOrder(sortKey, direction, jobPositionDefaultSort, string(types.ASC), jobPositionSortAllowlist)
}

var leaveTypeSortAllowlist = map[string]bool{
	"id":                   true,
	"name":                 true,
	"description":          true,
	"limit_amount":         true,
	"created_at":           true,
	"updated_at":           true,
	"is_paid":              true,
	"is_accrual":           true,
	"is_required_document": true,
}

const leaveTypeDefaultSort = "id"

func buildLeaveTypeOrderClause(sortKey, direction string) string {
	return buildAllowlistedOrder(sortKey, direction, leaveTypeDefaultSort, string(types.ASC), leaveTypeSortAllowlist)
}

var expenseTypeSortAllowlist = map[string]bool{
	"id":               true,
	"name":             true,
	"description":      true,
	"max_amount":       true,
	"active":           true,
	"requires_receipt": true,
	"role_name":        true,
	"created_at":       true,
	"updated_at":       true,
}

const expenseTypeDefaultSort = "name"

func buildExpenseTypeOrderClause(sortKey, direction string) (orderClause string, needsRoleJoin bool) {
	direction = types.NormalizeSortDirection(direction, string(types.ASC))
	key := types.AllowedSortOrDefault(sortKey, expenseTypeSortAllowlist, expenseTypeDefaultSort)

	et := domain.GetTableName("hr_expense_types")
	roles := domain.GetTableName("hr_roles")
	nulls := "NULLS LAST"

	switch key {
	case "role_name":
		return fmt.Sprintf("%s.name %s %s", roles, direction, nulls), true
	case "requires_receipt":
		return fmt.Sprintf("%s.requires_receipt %s", et, direction), false
	case "description":
		return fmt.Sprintf("%s.description %s", et, direction), false
	case "max_amount":
		return fmt.Sprintf("%s.max_amount %s %s", et, direction, nulls), false
	case "active":
		return fmt.Sprintf("%s.active %s", et, direction), false
	case "created_at":
		return fmt.Sprintf("%s.created_at %s", et, direction), false
	case "updated_at":
		return fmt.Sprintf("%s.updated_at %s", et, direction), false
	case "id":
		return fmt.Sprintf("%s.id %s", et, direction), false
	case "name":
		fallthrough
	default:
		return fmt.Sprintf("%s.name %s", et, direction), false
	}
}

var contractSortAllowlist = map[string]bool{
	"id":                    true,
	"contract_no":           true,
	"project_name":          true,
	"customer_contact_name": true,
	"start_date":            true,
	"end_date":              true,
	"status":                true,
	"created_at":            true,
	"updated_at":            true,
}

const contractDefaultSort = "created_at"

func buildContractOrderClause(sortKey, direction string) string {
	return buildAllowlistedOrder(sortKey, direction, contractDefaultSort, string(types.DESC), contractSortAllowlist)
}

var faqSortAllowlist = map[string]bool{
	"title":      true,
	"status":     true,
	"created_at": true,
	"updated_at": true,
}

const faqDefaultSort = "created_at"

func buildFAQOrderClause(sortKey, direction string) string {
	return buildAllowlistedOrder(sortKey, direction, faqDefaultSort, string(types.DESC), faqSortAllowlist)
}

var jobSortAllowlist = map[string]bool{
	"id":         true,
	"name":       true,
	"job_key":    true,
	"is_active":  true,
	"created_at": true,
	"updated_at": true,
}

const jobDefaultSort = "id"

func buildJobOrderClause(sortKey, direction string) string {
	return buildAllowlistedOrder(sortKey, direction, jobDefaultSort, string(types.ASC), jobSortAllowlist)
}

var jobHistorySortAllowlist = map[string]bool{
	"id":              true,
	"start_time":      true,
	"end_time":        true,
	"processed_count": true,
	"status":          true,
	"reference_date":  true,
}

const jobHistoryDefaultSort = "start_time"

// NormalizeJobHistorySortParams maps sort/direction to the values used in ORDER BY and page metadata.
func NormalizeJobHistorySortParams(sortParams types.SortParams) types.SortParams {
	sortParams.Direction = types.NormalizeSortDirection(sortParams.Direction, string(types.DESC))
	sortParams.Sort = types.AllowedSortOrDefault(sortParams.Sort, jobHistorySortAllowlist, jobHistoryDefaultSort)
	return sortParams
}

func buildJobHistoryOrderClause(sortKey, direction string) string {
	normalized := NormalizeJobHistorySortParams(types.SortParams{Sort: sortKey, Direction: direction})
	return fmt.Sprintf("%s %s", normalized.Sort, normalized.Direction)
}

var eventSortAllowlist = map[string]bool{
	"name":            true,
	"type":            true,
	"start_date":      true,
	"end_date":        true,
	"location":        true,
	"audience_filter": true,
	"status":          true,
}

const eventDefaultSort = "start_date"

func buildEventOrderClause(sortKey, direction string) string {
	direction = types.NormalizeSortDirection(direction, string(types.DESC))
	key := types.AllowedSortOrDefault(sortKey, eventSortAllowlist, eventDefaultSort)
	table := domain.GetTableName("events")
	return fmt.Sprintf("%s.%s %s NULLS LAST", table, key, direction)
}
