package authz_test

import (
	"testing"

	"kartezya-hr/internal/authz"
	"kartezya-hr/internal/domain"
)

func TestLeaveApproveCapabilities(t *testing.T) {
	if !authz.HasCapability([]string{domain.RoleAdmin}, authz.CanApproveLeave) {
		t.Fatal("ADMIN must be able to approve leave")
	}
	if !authz.HasCapability([]string{domain.RoleHR}, authz.CanApproveLeave) {
		t.Fatal("HR must be able to approve leave")
	}
	if authz.HasCapability([]string{domain.RoleFinance}, authz.CanApproveLeave) {
		t.Fatal("FINANCE must not be able to approve leave")
	}
	if authz.HasCapability([]string{domain.RoleEmployee}, authz.CanApproveLeave) {
		t.Fatal("EMPLOYEE must not be able to approve leave")
	}
}

func TestExpenseApproveAndPayCapabilities(t *testing.T) {
	if !authz.HasCapability([]string{domain.RoleAdmin}, authz.CanApproveExpense) {
		t.Fatal("ADMIN must be able to approve expense")
	}
	if !authz.HasCapability([]string{domain.RoleHR}, authz.CanApproveExpense) {
		t.Fatal("HR must be able to approve expense")
	}
	if authz.HasCapability([]string{domain.RoleFinance}, authz.CanApproveExpense) {
		t.Fatal("FINANCE must not be able to approve expense")
	}

	if !authz.HasCapability([]string{domain.RoleAdmin}, authz.CanPayExpense) {
		t.Fatal("ADMIN must be able to pay expense")
	}
	if !authz.HasCapability([]string{domain.RoleFinance}, authz.CanPayExpense) {
		t.Fatal("FINANCE must be able to pay expense")
	}
	if authz.HasCapability([]string{domain.RoleHR}, authz.CanPayExpense) {
		t.Fatal("HR must not be able to pay expense")
	}
}

func TestExpenseViewManagementCapabilities(t *testing.T) {
	for _, role := range []string{domain.RoleAdmin, domain.RoleHR, domain.RoleFinance} {
		if !authz.HasCapability([]string{role}, authz.CanViewExpenseManagement) {
			t.Fatalf("%s must be able to view expense management", role)
		}
	}
	if authz.HasCapability([]string{domain.RoleEmployee}, authz.CanViewExpenseManagement) {
		t.Fatal("EMPLOYEE must not view expense management")
	}
}

func TestExpenseTypeManagementCapabilities(t *testing.T) {
	if !authz.HasCapability([]string{domain.RoleAdmin}, authz.CanManageExpenseTypes) {
		t.Fatal("ADMIN must be able to manage expense types")
	}
	if !authz.HasCapability([]string{domain.RoleHR}, authz.CanManageExpenseTypes) {
		t.Fatal("HR must be able to manage expense types")
	}
	if !authz.HasCapability([]string{domain.RoleFinance}, authz.CanManageExpenseTypes) {
		t.Fatal("FINANCE must be able to manage expense types")
	}
	if authz.HasCapability([]string{domain.RoleEmployee}, authz.CanManageExpenseTypes) {
		t.Fatal("EMPLOYEE must not be able to manage expense types")
	}
}

func TestAccessAdminModulesCapabilities(t *testing.T) {
	if !authz.HasCapability([]string{domain.RoleAdmin}, authz.CanAccessAdminModules) {
		t.Fatal("ADMIN must be able to access admin modules")
	}
	if !authz.HasCapability([]string{domain.RoleHR}, authz.CanAccessAdminModules) {
		t.Fatal("HR must be able to access admin modules")
	}
	if authz.HasCapability([]string{domain.RoleFinance}, authz.CanAccessAdminModules) {
		t.Fatal("FINANCE must not be able to access admin modules")
	}
	if authz.HasCapability([]string{domain.RoleEmployee}, authz.CanAccessAdminModules) {
		t.Fatal("EMPLOYEE must not be able to access admin modules")
	}
	if authz.HasCapability([]string{domain.RoleHR}, authz.CanPayExpense) {
		t.Fatal("HR CanAccessAdminModules must not imply CanPayExpense")
	}
}

func TestEmployeeViewAndManageCapabilities(t *testing.T) {
	for _, role := range []string{domain.RoleAdmin, domain.RoleHR, domain.RoleFinance} {
		if !authz.HasCapability([]string{role}, authz.CanViewEmployees) {
			t.Fatalf("%s must be able to view employees", role)
		}
	}
	if authz.HasCapability([]string{domain.RoleEmployee}, authz.CanViewEmployees) {
		t.Fatal("EMPLOYEE must not view employees")
	}

	if !authz.HasCapability([]string{domain.RoleAdmin}, authz.CanManageEmployees) {
		t.Fatal("ADMIN must be able to manage employees")
	}
	if !authz.HasCapability([]string{domain.RoleHR}, authz.CanManageEmployees) {
		t.Fatal("HR must be able to manage employees")
	}
	if authz.HasCapability([]string{domain.RoleFinance}, authz.CanManageEmployees) {
		t.Fatal("FINANCE must not be able to manage employees")
	}
	if authz.HasCapability([]string{domain.RoleEmployee}, authz.CanManageEmployees) {
		t.Fatal("EMPLOYEE must not be able to manage employees")
	}
}
