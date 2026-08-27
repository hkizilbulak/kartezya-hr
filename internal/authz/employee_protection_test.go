package authz

import (
	"errors"
	"testing"

	"kartezya-hr/internal/domain"
)

func TestClassifyActor(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  ActorKind
	}{
		{name: "admin", roles: []string{domain.RoleAdmin}, want: ActorAdmin},
		{name: "hr", roles: []string{domain.RoleHR}, want: ActorHR},
		{name: "admin wins over hr", roles: []string{domain.RoleHR, domain.RoleAdmin}, want: ActorAdmin},
		{name: "finance", roles: []string{domain.RoleFinancial}, want: ActorOther},
		{name: "employee", roles: []string{domain.RoleEmployee}, want: ActorOther},
		{name: "empty", roles: nil, want: ActorOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyActor(tt.roles); got != tt.want {
				t.Fatalf("ClassifyActor(%v) = %v, want %v", tt.roles, got, tt.want)
			}
		})
	}
}

func TestValidateAssignableRoles(t *testing.T) {
	tests := []struct {
		name      string
		actor     ActorKind
		requested []string
		wantErr   error
	}{
		{
			name:      "admin can assign admin",
			actor:     ActorAdmin,
			requested: []string{domain.RoleAdmin},
		},
		{
			name:      "admin can assign all five",
			actor:     ActorAdmin,
			requested: []string{domain.RoleAdmin, domain.RoleEmployee, domain.RoleHR, domain.RoleFinancial, domain.RoleTeamLeader},
		},
		{
			name:      "hr can assign employee hr finance team_leader",
			actor:     ActorHR,
			requested: []string{domain.RoleEmployee, domain.RoleHR, domain.RoleFinancial, domain.RoleTeamLeader},
		},
		{
			name:      "hr cannot assign admin",
			actor:     ActorHR,
			requested: []string{domain.RoleEmployee, domain.RoleAdmin},
			wantErr:   ErrForbiddenAssignAdmin,
		},
		{
			name:      "hr blank role rejected",
			actor:     ActorHR,
			requested: []string{"  "},
			wantErr:   ErrInvalidRoleAssignment,
		},
		{
			name:      "hr unknown role rejected",
			actor:     ActorHR,
			requested: []string{"MANAGER"},
			wantErr:   ErrInvalidRoleAssignment,
		},
		{
			name:      "empty means no role change",
			actor:     ActorHR,
			requested: nil,
		},
		{
			name:      "admin unknown role rejected",
			actor:     ActorAdmin,
			requested: []string{"SUPERUSER"},
			wantErr:   ErrInvalidRoleAssignment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssignableRoles(tt.actor, tt.requested)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDenyHRMutatingAdminTarget(t *testing.T) {
	if err := DenyHRMutatingAdminTarget(ActorAdmin, true); err != nil {
		t.Fatalf("ADMIN must manage ADMIN target: %v", err)
	}
	if err := DenyHRMutatingAdminTarget(ActorHR, false); err != nil {
		t.Fatalf("HR must manage non-ADMIN target: %v", err)
	}
	if err := DenyHRMutatingAdminTarget(ActorHR, true); !errors.Is(err, ErrForbiddenAdminTarget) {
		t.Fatalf("HR must not manage ADMIN target, got %v", err)
	}
	if err := DenyHRMutatingAdminTarget(ActorOther, true); err != nil {
		t.Fatalf("ActorOther is not denied by this helper alone: %v", err)
	}
}

func TestFinanceCannotManageEmployees(t *testing.T) {
	if HasCapability([]string{domain.RoleFinancial}, CanManageEmployees) {
		t.Fatal("FINANCIAL must not mutate employees")
	}
}
