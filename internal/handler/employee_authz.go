package handler

import (
	"errors"
	"net/http"

	"kartezya-hr/internal/authz"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

// rejectIfHRMutatingAdminEmployee returns true when the request was rejected.
func rejectIfHRMutatingAdminEmployee(c *gin.Context, employeeService service.EmployeeService, employeeID uint, actorRoles []string) bool {
	actor := authz.ClassifyActor(actorRoles)
	if actor != authz.ActorHR {
		return false
	}

	targetHasAdmin, err := employeeService.EmployeeUserHasAdminRole(employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to verify target employee roles",
		})
		return true
	}

	if err := authz.DenyHRMutatingAdminTarget(actor, targetHasAdmin); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return true
	}
	return false
}

func mapEmployeeAuthzError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, authz.ErrForbiddenAdminTarget),
		errors.Is(err, authz.ErrForbiddenAssignAdmin):
		return http.StatusForbidden
	case errors.Is(err, authz.ErrInvalidRoleAssignment):
		return http.StatusBadRequest
	case err.Error() == "unauthorized to update this employee profile",
		err.Error() == "only administrators can delete employee profiles":
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
