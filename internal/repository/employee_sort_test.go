package repository

import (
	"strings"
	"testing"

	"kartezya-hr/internal/types"
)

func TestNormalizeEmployeeListSortKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "employee_name", want: "employee_name"},
		{in: "company_name", want: "company_name"},
		{in: "department_name", want: "department_name"},
		{in: "manager_name", want: "manager_name"},
		{in: "hire_date", want: "hire_date"},
		{in: "first_name", want: "employee_name"},
		{in: "manager", want: "manager_name"},
		{in: "employee_id", want: "employee_name"},
		{in: "", want: "employee_name"},
		{in: "'; DROP TABLE", want: "employee_name"},
	}

	for _, tt := range tests {
		if got := normalizeEmployeeListSortKey(tt.in); got != tt.want {
			t.Fatalf("normalizeEmployeeListSortKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildEmployeeListOrderClause(t *testing.T) {
	clause := buildEmployeeListOrderClause("employee_name", "desc")
	if !strings.Contains(clause, "first_name DESC") || !strings.Contains(clause, "last_name DESC") {
		t.Fatalf("employee_name clause missing name columns: %s", clause)
	}
	if strings.Contains(strings.ToLower(clause), "employee_id") && !strings.Contains(clause, "wi.employee_id") {
		t.Fatalf("employee_name must not sort by FK id: %s", clause)
	}

	company := buildEmployeeListOrderClause("company_name", "ASC")
	if !strings.Contains(company, "c.name") || !strings.Contains(company, "start_date DESC LIMIT 1") {
		t.Fatalf("company_name should use latest work-info company name subquery: %s", company)
	}
	if strings.Contains(company, "company_id ASC") || strings.Contains(company, "company_id DESC") {
		t.Fatalf("company_name must not sort by company_id: %s", company)
	}

	manager := buildEmployeeListOrderClause("manager_name", "asc")
	if !strings.Contains(manager, "d.manager") {
		t.Fatalf("manager_name should sort by department.manager: %s", manager)
	}

	invalid := buildEmployeeListOrderClause("'; DROP TABLE employees; --", "nope")
	if !strings.Contains(invalid, "first_name ASC") {
		t.Fatalf("invalid key/direction should use employee_name ASC default: %s", invalid)
	}
	if strings.Contains(invalid, "DROP TABLE") {
		t.Fatalf("raw input must not appear in ORDER BY: %s", invalid)
	}

	if dir := types.NormalizeSortDirection("desc", "ASC"); dir != "DESC" {
		t.Fatalf("expected DESC, got %s", dir)
	}
}
