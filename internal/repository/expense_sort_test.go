package repository

import (
	"strings"
	"testing"

	"kartezya-hr/internal/domain"
)

func TestNormalizeExpenseRequestSortKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "employee_name", want: "employee_name"},
		{in: "expense_type_name", want: "expense_type_name"},
		{in: "description", want: "description"},
		{in: "amount", want: "amount"},
		{in: "expense_date", want: "expense_date"},
		{in: "created_at", want: "created_at"},
		{in: "employee_id", want: "employee_name"},
		{in: "expense_type_id", want: "expense_type_name"},
		{in: "status", want: "created_at"},
		{in: "1; DROP TABLE", want: "created_at"},
		{in: "", want: "created_at"},
	}

	for _, tt := range tests {
		if got := normalizeExpenseRequestSortKey(tt.in); got != tt.want {
			t.Fatalf("normalizeExpenseRequestSortKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildExpenseRequestOrderClause(t *testing.T) {
	empClause, needEmp, needType := buildExpenseRequestOrderClause("employee_name", "asc")
	if !needEmp || needType {
		t.Fatalf("employee_name should require employee join only")
	}
	if !strings.Contains(empClause, "first_name ASC") || !strings.Contains(empClause, "last_name ASC") {
		t.Fatalf("employee_name should sort by first/last name: %s", empClause)
	}
	if strings.Contains(empClause, "employee_id ASC") || strings.Contains(empClause, "employee_id DESC") {
		t.Fatalf("must not sort by employee_id: %s", empClause)
	}

	typeClause, needEmp, needType := buildExpenseRequestOrderClause("expense_type_name", "DESC")
	if needEmp || !needType {
		t.Fatalf("expense_type_name should require type join only")
	}
	if !strings.Contains(typeClause, ".name DESC") {
		t.Fatalf("expense_type_name should sort by type name: %s", typeClause)
	}

	amount, _, _ := buildExpenseRequestOrderClause("amount", "desc")
	if !strings.Contains(amount, "amount DESC") {
		t.Fatalf("amount clause unexpected: %s", amount)
	}

	invalid, needEmp, needType := buildExpenseRequestOrderClause("'; evil", "sideways")
	if needEmp || needType {
		t.Fatalf("invalid key should use created_at default without joins")
	}
	if !strings.Contains(invalid, "created_at DESC") {
		t.Fatalf("invalid key should default created_at DESC: %s", invalid)
	}
	if strings.Contains(invalid, "evil") {
		t.Fatalf("raw input must not appear in ORDER BY: %s", invalid)
	}
}

func TestExpenseRequestListFiltersQualifySharedColumns(t *testing.T) {
	exp := domain.GetTableName("hr_expense_requests")
	exprs := expenseRequestListFilterExpressions(exp)

	shared := []string{"deleted", "employee_id", "status", "expense_type_id", "expense_date"}
	for _, col := range shared {
		found := false
		for _, expr := range exprs {
			qualified := exp + "." + col
			if strings.Contains(expr, qualified) {
				found = true
				// Bare column reference without table prefix must not appear as a WHERE key.
				if strings.Contains(expr, " "+col+" ") || strings.HasPrefix(expr, col+" ") {
					t.Fatalf("unqualified column %q in filter expression %q", col, expr)
				}
			}
		}
		if !found {
			t.Fatalf("expected qualified filter for %s.%s among %v", exp, col, exprs)
		}
	}

	// Joined sorts must still request joins (regression: do not "fix" ambiguity by dropping joins).
	_, needEmp, _ := buildExpenseRequestOrderClause("employee_name", "ASC")
	if !needEmp {
		t.Fatal("employee_name sort must keep employee join")
	}
	_, _, needType := buildExpenseRequestOrderClause("expense_type_name", "ASC")
	if !needType {
		t.Fatal("expense_type_name sort must keep expense type join")
	}
}
