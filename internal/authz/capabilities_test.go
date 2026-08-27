package authz

import (
	"testing"

	"kartezya-hr/internal/domain"
)

func TestHasCapability(t *testing.T) {
	tests := []struct {
		name       string
		roles      []string
		capability Capability
		want       bool
	}{
		{
			name:       "admin can pay expense",
			roles:      []string{domain.RoleAdmin},
			capability: CanPayExpense,
			want:       true,
		},
		{
			name:       "hr cannot pay expense",
			roles:      []string{domain.RoleHR},
			capability: CanPayExpense,
			want:       false,
		},
		{
			name:       "hr can approve leave",
			roles:      []string{domain.RoleHR},
			capability: CanApproveLeave,
			want:       true,
		},
		{
			name:       "finance can pay expense",
			roles:      []string{domain.RoleFinancial},
			capability: CanPayExpense,
			want:       true,
		},
		{
			name:       "finance cannot approve expense",
			roles:      []string{domain.RoleFinancial},
			capability: CanApproveExpense,
			want:       false,
		},
		{
			name:       "finance cannot manage employees",
			roles:      []string{domain.RoleFinancial},
			capability: CanManageEmployees,
			want:       false,
		},
		{
			name:       "finance can view employees",
			roles:      []string{domain.RoleFinancial},
			capability: CanViewEmployees,
			want:       true,
		},
		{
			name:       "employee cannot view employees",
			roles:      []string{domain.RoleEmployee},
			capability: CanViewEmployees,
			want:       false,
		},
		{
			name:       "hr can manage expense types",
			roles:      []string{domain.RoleHR},
			capability: CanManageExpenseTypes,
			want:       true,
		},
		{
			name:       "finance can manage expense types",
			roles:      []string{domain.RoleFinancial},
			capability: CanManageExpenseTypes,
			want:       true,
		},
		{
			name:       "employee has no management capabilities",
			roles:      []string{domain.RoleEmployee},
			capability: CanViewLeaveManagement,
			want:       false,
		},
		{
			name:       "admin can access admin modules",
			roles:      []string{domain.RoleAdmin},
			capability: CanAccessAdminModules,
			want:       true,
		},
		{
			name:       "hr can access admin modules",
			roles:      []string{domain.RoleHR},
			capability: CanAccessAdminModules,
			want:       true,
		},
		{
			name:       "finance cannot access admin modules",
			roles:      []string{domain.RoleFinancial},
			capability: CanAccessAdminModules,
			want:       false,
		},
		{
			name:       "employee cannot access admin modules",
			roles:      []string{domain.RoleEmployee},
			capability: CanAccessAdminModules,
			want:       false,
		},
		{
			name:       "team leader can manage cv",
			roles:      []string{domain.RoleTeamLeader},
			capability: CanManageCV,
			want:       true,
		},
		{
			name:       "team leader can manage academy",
			roles:      []string{domain.RoleTeamLeader},
			capability: CanManageAcademy,
			want:       true,
		},
		{
			name:       "team leader cannot access admin modules",
			roles:      []string{domain.RoleTeamLeader},
			capability: CanAccessAdminModules,
			want:       false,
		},
		{
			name:       "hr admin modules does not imply pay",
			roles:      []string{domain.RoleHR},
			capability: CanPayExpense,
			want:       false,
		},
		{
			name:       "multi-role union grants pay from finance",
			roles:      []string{domain.RoleHR, domain.RoleFinancial},
			capability: CanPayExpense,
			want:       true,
		},
		{
			name:       "multi-role union grants approve from hr",
			roles:      []string{domain.RoleHR, domain.RoleFinancial},
			capability: CanApproveExpense,
			want:       true,
		},
		{
			name:       "empty roles",
			roles:      []string{},
			capability: CanManageEmployees,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCapability(tt.roles, tt.capability); got != tt.want {
				t.Fatalf("HasCapability(%v, %q) = %v, want %v", tt.roles, tt.capability, got, tt.want)
			}
		})
	}
}

func TestAdminHasAllTicketCapabilities(t *testing.T) {
	all := []Capability{
		CanViewEmployees,
		CanManageEmployees,
		CanManageOrgMaster,
		CanManageLeaveTypes,
		CanViewLeaveManagement,
		CanApproveLeave,
		CanViewExpenseManagement,
		CanApproveExpense,
		CanPayExpense,
		CanManageExpenseTypes,
		CanManageOtherRequests,
		CanManageRequestTypes,
		CanAccessAdminModules,
		CanManageAcademy,
		CanManageCV,
	}
	for _, cap := range all {
		if !HasCapability([]string{domain.RoleAdmin}, cap) {
			t.Fatalf("ADMIN missing capability %q", cap)
		}
	}
}
