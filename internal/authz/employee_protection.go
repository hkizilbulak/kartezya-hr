package authz

import (
	"errors"
	"fmt"
	"strings"

	"kartezya-hr/internal/domain"
)

// ActorKind classifies the caller for employee-management authorization.
type ActorKind int

const (
	// ActorOther is any caller that is neither ADMIN nor HR (e.g. FINANCIAL, EMPLOYEE).
	ActorOther ActorKind = iota
	// ActorAdmin is a caller with the ADMIN role (takes precedence over HR).
	ActorAdmin
	// ActorHR is a caller with HR but not ADMIN.
	ActorHR
)

var (
	// ErrForbiddenAdminTarget is returned when HR attempts to mutate an ADMIN user.
	ErrForbiddenAdminTarget = errors.New("forbidden: cannot modify an ADMIN user")
	// ErrForbiddenAssignAdmin is returned when HR attempts to assign the ADMIN role.
	ErrForbiddenAssignAdmin = errors.New("forbidden: cannot assign ADMIN role")
	// ErrInvalidRoleAssignment is returned for blank or unsupported role values.
	ErrInvalidRoleAssignment = errors.New("invalid role assignment")
)

// ClassifyActor returns the employee-management actor kind for the given JWT roles.
// ADMIN wins when both ADMIN and HR are present.
func ClassifyActor(roles []string) ActorKind {
	for _, role := range roles {
		if role == domain.RoleAdmin {
			return ActorAdmin
		}
	}
	for _, role := range roles {
		if role == domain.RoleHR {
			return ActorHR
		}
	}
	return ActorOther
}

// ContainsAdminRole reports whether the role list includes ADMIN.
func ContainsAdminRole(roles []string) bool {
	for _, role := range roles {
		if strings.TrimSpace(role) == domain.RoleAdmin {
			return true
		}
	}
	return false
}

// DenyHRMutatingAdminTarget returns ErrForbiddenAdminTarget when an HR actor
// attempts to mutate a target that currently has the ADMIN role.
// ADMIN actors are never denied by this check.
func DenyHRMutatingAdminTarget(actor ActorKind, targetHasAdmin bool) error {
	if actor == ActorHR && targetHasAdmin {
		return ErrForbiddenAdminTarget
	}
	return nil
}

// ValidateAssignableRoles validates a role-update payload for the given actor.
// An empty requested list means "do not change roles" and is allowed.
// Multi-role payloads are supported for allowed role values.
func ValidateAssignableRoles(actor ActorKind, requested []string) error {
	if len(requested) == 0 {
		return nil
	}

	allowed := map[string]bool{
		domain.RoleEmployee: true,
		domain.RoleHR:       true,
		domain.RoleFinancial: true,
	}
	if actor == ActorAdmin {
		allowed[domain.RoleAdmin] = true
	}

	seen := make(map[string]bool, len(requested))
	for _, raw := range requested {
		role := strings.TrimSpace(raw)
		if role == "" {
			return fmt.Errorf("%w: role value cannot be blank", ErrInvalidRoleAssignment)
		}
		if actor != ActorAdmin && role == domain.RoleAdmin {
			return ErrForbiddenAssignAdmin
		}
		if !allowed[role] {
			return fmt.Errorf("%w: unsupported role %q", ErrInvalidRoleAssignment, role)
		}
		if seen[role] {
			continue
		}
		seen[role] = true
	}

	return nil
}
