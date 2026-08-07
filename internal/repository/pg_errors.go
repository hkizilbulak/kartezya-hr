package repository

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation reports whether err is a PostgreSQL unique_violation (SQLSTATE 23505).
// Uses typed pgconn.PgError; does not parse error message strings.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// employeeGradeActiveUniqueIndexSuffix is the stable suffix of
// ux_<prefix>_employee_grades_employee_id_status_active (prefix-agnostic match).
const employeeGradeActiveUniqueIndexSuffix = "employee_id_status_active"

// IsEmployeeGradeActiveUniqueViolation reports a unique violation on the partial
// unique index that enforces one ACTIVE (deleted=false) row per employee.
// Other unique violations (e.g. primary key / sequence desync) must NOT be mapped
// to ErrEmployeeGradeActiveConflict.
func IsEmployeeGradeActiveUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	name := pgErr.ConstraintName
	if name == "" {
		name = pgErr.Message
	}
	return strings.Contains(name, employeeGradeActiveUniqueIndexSuffix)
}
