package service

import "testing"

func TestAuthorizeExpenseAccess(t *testing.T) {
	tests := []struct {
		name              string
		expenseEmployeeID uint
		callerEmployeeID  uint
		canViewManagement bool
		wantErr           bool
	}{
		{name: "owner allowed", expenseEmployeeID: 5, callerEmployeeID: 5, canViewManagement: false, wantErr: false},
		{name: "unrelated employee denied", expenseEmployeeID: 5, callerEmployeeID: 9, canViewManagement: false, wantErr: true},
		{name: "management viewer allowed", expenseEmployeeID: 5, callerEmployeeID: 9, canViewManagement: true, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizeExpenseAccess(tt.expenseEmployeeID, tt.callerEmployeeID, tt.canViewManagement)
			if tt.wantErr && err == nil {
				t.Fatal("expected access denied")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
