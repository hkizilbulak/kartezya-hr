package repository

import (
	"strings"
	"testing"
)

func TestNormalizeOtherRequestSortKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "employee_name", want: "employee_name"},
		{in: "request_type_name", want: "request_type_name"},
		{in: "request_type", want: "request_type_name"},
		{in: "employee_id", want: "employee_name"},
		{in: "request_type_id", want: "request_type_name"},
		{in: "description", want: "description"},
		{in: "created_at", want: "created_at"},
		{in: "status", want: "created_at"},
		{in: "", want: "created_at"},
	}
	for _, tt := range tests {
		if got := normalizeOtherRequestSortKey(tt.in); got != tt.want {
			t.Fatalf("normalizeOtherRequestSortKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildOtherRequestOrderClause(t *testing.T) {
	emp, needEmp, needType := buildOtherRequestOrderClause("employee_name", "desc")
	if !needEmp || needType {
		t.Fatalf("employee_name should require employee join only")
	}
	if !strings.Contains(emp, "first_name DESC") || !strings.Contains(emp, "last_name DESC") {
		t.Fatalf("unexpected employee clause: %s", emp)
	}

	typ, needEmp, needType := buildOtherRequestOrderClause("request_type_name", "ASC")
	if needEmp || !needType {
		t.Fatalf("request_type_name should require type join only")
	}
	if !strings.Contains(typ, ".name ASC") {
		t.Fatalf("unexpected type clause: %s", typ)
	}

	invalid, needEmp, needType := buildOtherRequestOrderClause("bad", "nope")
	if needEmp || needType || !strings.Contains(invalid, "created_at ASC") {
		t.Fatalf("invalid should default created_at ASC without joins: %s", invalid)
	}
}
