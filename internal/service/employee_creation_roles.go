package service

import (
	"errors"
	"strings"

	"kartezya-hr/internal/domain"
)

// ValidateEmployeeCreationRoles requires exactly one allowed creation role.
// ADMIN and unknown roles are rejected.
func ValidateEmployeeCreationRoles(roles []string) (string, error) {
	if len(roles) == 0 {
		return "", errors.New("role is required")
	}
	if len(roles) != 1 {
		return "", errors.New("exactly one role must be provided")
	}

	role := strings.TrimSpace(roles[0])
	if role == "" {
		return "", errors.New("role is required")
	}

	switch role {
	case domain.RoleEmployee, domain.RoleHR, domain.RoleFinance:
		return role, nil
	case domain.RoleAdmin:
		return "", errors.New("ADMIN role cannot be assigned during employee creation")
	default:
		return "", errors.New("unsupported role")
	}
}
