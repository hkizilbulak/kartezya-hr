package service

import (
	"testing"

	"kartezya-hr/internal/domain"
)

func TestValidateEmployeeCreationRoles(t *testing.T) {
	tests := []struct {
		name    string
		roles   []string
		want    string
		wantErr string
	}{
		{
			name:  "employee accepted",
			roles: []string{domain.RoleEmployee},
			want:  domain.RoleEmployee,
		},
		{
			name:  "hr accepted",
			roles: []string{domain.RoleHR},
			want:  domain.RoleHR,
		},
		{
			name:  "finance accepted",
			roles: []string{domain.RoleFinancial},
			want:  domain.RoleFinancial,
		},
		{
			name:  "team leader accepted",
			roles: []string{domain.RoleTeamLeader},
			want:  domain.RoleTeamLeader,
		},
		{
			name:    "admin rejected",
			roles:   []string{domain.RoleAdmin},
			wantErr: "ADMIN role cannot be assigned during employee creation",
		},
		{
			name:    "missing role rejected",
			roles:   nil,
			wantErr: "role is required",
		},
		{
			name:    "empty slice rejected",
			roles:   []string{},
			wantErr: "role is required",
		},
		{
			name:    "blank role rejected",
			roles:   []string{"  "},
			wantErr: "role is required",
		},
		{
			name:    "unknown role rejected",
			roles:   []string{"MANAGER"},
			wantErr: "unsupported role",
		},
		{
			name:    "multiple roles rejected",
			roles:   []string{domain.RoleEmployee, domain.RoleHR},
			wantErr: "exactly one role must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateEmployeeCreationRoles(tt.roles)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
