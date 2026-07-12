package repository

import (
	"strings"
	"testing"
	"time"
)

func TestBuildLeaveListGroupClauseSQL(t *testing.T) {
	pending, args, ok := buildLeaveListGroupClause("pending", "lr")
	if !ok {
		t.Fatal("pending should apply")
	}
	if !strings.Contains(pending, "start_date::date > CURRENT_DATE") {
		t.Fatalf("pending should require future start_date: %s", pending)
	}
	if strings.Contains(pending, "IS NULL") {
		t.Fatalf("pending must not include NULL start_date: %s", pending)
	}
	if len(args) != 2 || args[0] != "PENDING" || args[1] != "APPROVED" {
		t.Fatalf("unexpected pending args: %#v", args)
	}

	completed, args, ok := buildLeaveListGroupClause("completed", "lr")
	if !ok {
		t.Fatal("completed should apply")
	}
	if !strings.Contains(completed, "start_date IS NULL") {
		t.Fatalf("completed must include NULL start_date: %s", completed)
	}
	if !strings.Contains(completed, "start_date::date <= CURRENT_DATE") {
		t.Fatalf("completed should include today/past start_date: %s", completed)
	}
	if len(args) != 3 || args[0] != "REJECTED" || args[1] != "CANCELLED" || args[2] != "APPROVED" {
		t.Fatalf("unexpected completed args: %#v", args)
	}

	if _, _, ok := buildLeaveListGroupClause("", "lr"); ok {
		t.Fatal("empty list_group should not apply a filter")
	}
}

func TestLeaveListGroupMembership(t *testing.T) {
	today := time.Date(2026, 7, 12, 15, 30, 0, 0, time.UTC)
	future := today.AddDate(0, 0, 3)
	past := today.AddDate(0, 0, -3)
	sameDay := time.Date(2026, 7, 12, 8, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		status    string
		startDate *time.Time
		wantPend  bool
		wantComp  bool
	}{
		{name: "APPROVED future → pending only", status: "APPROVED", startDate: &future, wantPend: true, wantComp: false},
		{name: "APPROVED today → completed only", status: "APPROVED", startDate: &sameDay, wantPend: false, wantComp: true},
		{name: "APPROVED past → completed only", status: "APPROVED", startDate: &past, wantPend: false, wantComp: true},
		{name: "APPROVED NULL → completed only", status: "APPROVED", startDate: nil, wantPend: false, wantComp: true},
		{name: "PENDING → pending only", status: "PENDING", startDate: &future, wantPend: true, wantComp: false},
		{name: "REJECTED → completed only", status: "REJECTED", startDate: &future, wantPend: false, wantComp: true},
		{name: "CANCELLED → completed only", status: "CANCELLED", startDate: &future, wantPend: false, wantComp: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gotPend := matchesLeaveListGroup("pending", tt.status, tt.startDate, today)
			gotComp := matchesLeaveListGroup("completed", tt.status, tt.startDate, today)
			if gotPend != tt.wantPend || gotComp != tt.wantComp {
				t.Fatalf("pending=%v want %v; completed=%v want %v", gotPend, tt.wantPend, gotComp, tt.wantComp)
			}
			if gotPend == gotComp {
				t.Fatalf("groups must be mutually exclusive; both=%v", gotPend)
			}
			if !gotPend && !gotComp {
				t.Fatal("request must belong to exactly one group")
			}
		})
	}
}
