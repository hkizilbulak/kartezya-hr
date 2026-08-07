package domain

import "errors"

// Sentinel errors for employee-grade assign / lifecycle validation.
var (
	ErrEmployeeGradeNotFound = errors.New("employee grade not found")

	ErrEmployeeGradeEmployeeNotFound = errors.New("employee not found")

	ErrEmployeeGradeGradeNotFound = errors.New("grade not found")

	ErrEmployeeGradeInvalidStartDate = errors.New("invalid employee grade start date")

	// ErrEmployeeGradeInvalidCloseDate is returned when new start_date cannot produce
	// a valid end_date (newStart-1) that is >= the active grade's start_date.
	ErrEmployeeGradeInvalidCloseDate = errors.New(
		"new start date must be at least one day after the active grade start date",
	)

	ErrEmployeeGradeDuplicateAssignment = errors.New(
		"duplicate employee grade assignment for the same employee, grade and start date",
	)

	ErrEmployeeGradeActiveConflict = errors.New(
		"another active grade already exists for this employee",
	)

	ErrEmployeeGradeActiveCannotDelete = errors.New(
		"active employee grade cannot be deleted; assign a new grade first",
	)

	ErrEmployeeGradeActiveUpdateForbidden = errors.New(
		"active employee grade cannot be updated via this endpoint; assign a new grade instead",
	)

	ErrEmployeeGradeEmployeeImmutable = errors.New(
		"employee_id cannot be changed on an employee grade record",
	)

	ErrEmployeeGradeInactiveRequiresEndDate = errors.New(
		"inactive employee grade requires an end date",
	)

	ErrEmployeeGradeEndBeforeStart = errors.New(
		"end date must be on or after start date",
	)
)
