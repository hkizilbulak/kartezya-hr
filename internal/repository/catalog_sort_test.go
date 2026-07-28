package repository

import (
	"strings"
	"testing"

	"kartezya-hr/internal/types"
)

func TestBuildAllowlistedOrderInvalidNeverReachesSQL(t *testing.T) {
	evil := "id; DROP TABLE x"
	got := buildAllowlistedOrder(evil, "sideways", "name", "ASC", map[string]bool{"name": true, "created_at": true})
	if strings.Contains(got, "DROP") || strings.Contains(got, evil) {
		t.Fatalf("raw input leaked into ORDER BY: %s", got)
	}
	if got != "name ASC" {
		t.Fatalf("expected name ASC, got %s", got)
	}
	if dir := types.NormalizeSortDirection("DeSc", "ASC"); dir != "DESC" {
		t.Fatalf("mixed-case direction: got %s", dir)
	}
}

func TestLeaveTypeOrderClause(t *testing.T) {
	if got := buildLeaveTypeOrderClause("limit_amount", "desc"); got != "limit_amount DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildLeaveTypeOrderClause("is_paid", "ASC"); got != "is_paid ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildLeaveTypeOrderClause("is_accrual", "desc"); got != "is_accrual DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildLeaveTypeOrderClause("is_required_document", "ASC"); got != "is_required_document ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildLeaveTypeOrderClause("status", "ASC"); got != "id ASC" {
		t.Fatalf("invalid should default id ASC, got %s", got)
	}
}

func TestExpenseTypeOrderClause(t *testing.T) {
	clause, needRole := buildExpenseTypeOrderClause("max_amount", "ASC")
	if needRole || !strings.Contains(clause, "max_amount ASC") {
		t.Fatalf("got %s needRole=%v", clause, needRole)
	}
	clause, needRole = buildExpenseTypeOrderClause("active", "desc")
	if needRole || !strings.Contains(clause, "active DESC") {
		t.Fatalf("got %s needRole=%v", clause, needRole)
	}
	clause, needRole = buildExpenseTypeOrderClause("requires_receipt", "ASC")
	if needRole || !strings.Contains(clause, "requires_receipt ASC") {
		t.Fatalf("got %s needRole=%v", clause, needRole)
	}
	clause, needRole = buildExpenseTypeOrderClause("role_name", "DESC")
	if !needRole || !strings.Contains(clause, ".name DESC") {
		t.Fatalf("role_name should join roles and sort by name: %s needRole=%v", clause, needRole)
	}
	if strings.Contains(clause, "role_id") {
		t.Fatalf("must not sort by role_id: %s", clause)
	}
	clause, needRole = buildExpenseTypeOrderClause("bad", "nope")
	if needRole || !strings.Contains(clause, ".name ASC") {
		t.Fatalf("invalid should default name ASC, got %s needRole=%v", clause, needRole)
	}
}

func TestFAQOrderClauseStatus(t *testing.T) {
	if got := buildFAQOrderClause("status", "asc"); got != "status ASC" {
		t.Fatalf("got %s", got)
	}
}

func TestCompanyOrderClauseAddress(t *testing.T) {
	if got := buildCompanyOrderClause("address", "DESC"); got != "address DESC" {
		t.Fatalf("got %s", got)
	}
}

func TestContractOrderClause(t *testing.T) {
	if got := buildContractOrderClause("contract_no", "asc"); got != "contract_no ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildContractOrderClause("status", "DESC"); got != "status DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildContractOrderClause("hack", "x"); got != "created_at DESC" {
		t.Fatalf("invalid should default created_at DESC, got %s", got)
	}
}

func TestJobOrderClause(t *testing.T) {
	if got := buildJobOrderClause("job_key", "ASC"); got != "job_key ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobOrderClause("is_active", "desc"); got != "is_active DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobOrderClause("cron_expression", "ASC"); got != "id ASC" {
		t.Fatalf("invalid should default id ASC, got %s", got)
	}
}

func TestJobHistoryOrderClause(t *testing.T) {
	if got := buildJobHistoryOrderClause("start_time", "ASC"); got != "start_time ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobHistoryOrderClause("end_time", "desc"); got != "end_time DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobHistoryOrderClause("processed_count", "ASC"); got != "processed_count ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobHistoryOrderClause("status", "DESC"); got != "status DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobHistoryOrderClause("reference_date", "ASC"); got != "reference_date ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobHistoryOrderClause("reference_date", "desc"); got != "reference_date DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobHistoryOrderClause("id", "ASC"); got != "id ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildJobHistoryOrderClause("hack; DROP TABLE x", "nope"); got != "start_time DESC" {
		t.Fatalf("invalid should default start_time DESC, got %s", got)
	}
	if strings.Contains(buildJobHistoryOrderClause("id; DROP TABLE x", "ASC"), "DROP") {
		t.Fatal("raw input leaked into ORDER BY")
	}
}

func TestNormalizeJobHistorySortParams(t *testing.T) {
	got := NormalizeJobHistorySortParams(types.SortParams{Sort: "invalid_field", Direction: "wrong"})
	if got.Sort != "start_time" || got.Direction != "DESC" {
		t.Fatalf("invalid input: got %+v", got)
	}
	got = NormalizeJobHistorySortParams(types.SortParams{})
	if got.Sort != "start_time" || got.Direction != "DESC" {
		t.Fatalf("empty input: got %+v", got)
	}
	got = NormalizeJobHistorySortParams(types.SortParams{Sort: "processed_count", Direction: "asc"})
	if got.Sort != "processed_count" || got.Direction != "ASC" {
		t.Fatalf("valid input: got %+v", got)
	}
	got = NormalizeJobHistorySortParams(types.SortParams{Sort: "reference_date", Direction: "DESC"})
	if got.Sort != "reference_date" || got.Direction != "DESC" {
		t.Fatalf("reference_date input: got %+v", got)
	}
}

func TestCompanyDepartmentJobPositionOrderClause(t *testing.T) {
	if got := buildCompanyOrderClause("email", "DESC"); got != "email DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildCompanyOrderClause("bad", "asc"); got != "id ASC" {
		t.Fatalf("got %s", got)
	}
	dept := buildDepartmentOrderClause("company_name", "asc")
	if !strings.Contains(dept, ".name ASC") {
		t.Fatalf("company_name should sort company.name: %s", dept)
	}
	if got := buildJobPositionOrderClause("title", "DESC"); got != "title DESC" {
		t.Fatalf("got %s", got)
	}
}

func TestFAQAndEventOrderClause(t *testing.T) {
	if got := buildFAQOrderClause("title", "asc"); got != "title ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildFAQOrderClause("nope", "x"); got != "created_at DESC" {
		t.Fatalf("got %s", got)
	}
	ev := buildEventOrderClause("audience_filter", "asc")
	if !strings.Contains(ev, "audience_filter ASC") || !strings.Contains(ev, "NULLS LAST") {
		t.Fatalf("unexpected event clause: %s", ev)
	}
	if got := buildEventOrderClause("evil", "DESC"); !strings.Contains(got, "start_date DESC") {
		t.Fatalf("invalid event key should default start_date: %s", got)
	}
}
