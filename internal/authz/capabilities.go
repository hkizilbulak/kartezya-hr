package authz

import "kartezya-hr/internal/domain"

// Capability identifies an operation-level permission.
type Capability string

const (
	CanViewEmployees         Capability = "canViewEmployees"
	CanManageEmployees       Capability = "canManageEmployees"
	CanManageOrgMaster       Capability = "canManageOrgMaster"
	CanManageLeaveTypes      Capability = "canManageLeaveTypes"
	CanViewLeaveManagement   Capability = "canViewLeaveManagement"
	CanApproveLeave          Capability = "canApproveLeave"
	CanViewExpenseManagement Capability = "canViewExpenseManagement"
	CanApproveExpense        Capability = "canApproveExpense"
	CanPayExpense            Capability = "canPayExpense"
	CanManageExpenseTypes    Capability = "canManageExpenseTypes"
	CanManageOtherRequests   Capability = "canManageOtherRequests"
	CanManageRequestTypes    Capability = "canManageRequestTypes"
	CanAccessAdminModules    Capability = "canAccessAdminModules"
)

// RoleCapabilities maps each role to the capabilities granted by this ticket.
// EMPLOYEE has no management capabilities; self-service remains JWT + ownership.
var RoleCapabilities = map[string][]Capability{
	domain.RoleAdmin: {
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
	},
	domain.RoleHR: {
		CanViewEmployees,
		CanManageEmployees,
		CanManageOrgMaster,
		CanManageLeaveTypes,
		CanViewLeaveManagement,
		CanApproveLeave,
		CanViewExpenseManagement,
		CanApproveExpense,
		CanManageExpenseTypes,
		CanManageOtherRequests,
		CanManageRequestTypes,
		CanAccessAdminModules,
	},
	domain.RoleFinancial: {
		CanViewEmployees,
		CanViewExpenseManagement,
		CanPayExpense,
		CanManageExpenseTypes,
	},
	domain.RoleEmployee: {},
}

// HasCapability reports whether any of the given roles grants the capability.
func HasCapability(roles []string, capability Capability) bool {
	for _, role := range roles {
		for _, cap := range RoleCapabilities[role] {
			if cap == capability {
				return true
			}
		}
	}
	return false
}
