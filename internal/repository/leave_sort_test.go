package repository

import (
	"strings"
	"testing"
)

func TestNormalizeLeaveRequestSortKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "employee_name", want: "employee_name"},
		{in: "leave_type_name", want: "leave_type_name"},
		{in: "leave_type", want: "leave_type_name"},
		{in: "employee_id", want: "employee_name"},
		{in: "leave_type_id", want: "leave_type_name"},
		{in: "start_date", want: "start_date"},
		{in: "end_date", want: "end_date"},
		{in: "requested_days", want: "requested_days"},
		{in: "created_at", want: "created_at"},
		{in: "status", want: "created_at"},
		{in: "1; DROP TABLE", want: "created_at"},
		{in: "", want: "created_at"},
	}
	for _, tt := range tests {
		if got := normalizeLeaveRequestSortKey(tt.in); got != tt.want {
			t.Fatalf("normalizeLeaveRequestSortKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildLeaveRequestOrderClause(t *testing.T) {
	emp, needEmp, needType := buildLeaveRequestOrderClause("employee_name", "asc")
	if !needEmp || needType {
		t.Fatalf("employee_name should require employee join only")
	}
	if !strings.Contains(emp, "first_name ASC") || !strings.Contains(emp, "last_name ASC") {
		t.Fatalf("employee_name should sort first then last: %s", emp)
	}
	if strings.Contains(emp, "employee_id ASC") || strings.Contains(emp, "employee_id DESC") {
		t.Fatalf("must not sort by employee_id: %s", emp)
	}

	typ, needEmp, needType := buildLeaveRequestOrderClause("leave_type_name", "DESC")
	if needEmp || !needType {
		t.Fatalf("leave_type_name should require type join only")
	}
	if !strings.Contains(typ, ".name DESC") {
		t.Fatalf("leave_type_name should sort by type name: %s", typ)
	}

	invalid, needEmp, needType := buildLeaveRequestOrderClause("'; evil", "sideways")
	if needEmp || needType {
		t.Fatalf("invalid key should default without joins")
	}
	if !strings.Contains(invalid, "created_at DESC") {
		t.Fatalf("invalid should default created_at DESC: %s", invalid)
	}
	if strings.Contains(invalid, "evil") {
		t.Fatalf("raw input must not appear in ORDER BY: %s", invalid)
	}
}
