package repository

import (
	"fmt"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

// Active EmployeeGrade is the current-grade source of truth for dashboard counts,
// Employees grade_id filtering (Phase 3), and grade/work-day report "current grade"
// columns (Phase 6).
//
// Predicate (identical everywhere):
//   deleted = false AND status = ACTIVE
//
// Notes:
//   - end_date IS NULL is enforced by DB CHECK after status migration.
//   - start_date <= CURRENT_DATE is intentionally NOT part of this predicate:
//     Phase 2 assign allows future start_date and immediately makes that row ACTIVE
//     (closing the previous ACTIVE). Dashboard, Employees, and reports must share this rule.
//   - Requires hr_employee_grades.status column (EnsureEmployeeGradeStatusConstraints /
//     migrate_employee_grade_status.sql). Do not assume AutoMigrate alone without Ensure.
//   - employees.grade_id is ignored for filtering/counting/report current grade.

// activeEmployeeGradeSQLConditions is the shared WHERE fragment for an employee_grades alias.
func activeEmployeeGradeSQLConditions(egAlias string) string {
	return fmt.Sprintf(
		"%s.deleted = false AND %s.status = '%s'",
		egAlias,
		egAlias,
		domain.EmployeeGradeStatusActive,
	)
}

// ActiveEmployeeGradeJoinOn builds a LEFT/INNER JOIN ON clause linking eg → employee.
// Example: ActiveEmployeeGradeJoinOn("eg", employeesTable+".id")
func ActiveEmployeeGradeJoinOn(egAlias, employeeIDExpr string) string {
	return fmt.Sprintf(
		"%s.employee_id = %s AND %s",
		egAlias,
		employeeIDExpr,
		activeEmployeeGradeSQLConditions(egAlias),
	)
}

// buildActiveCurrentGradeSelectSQL returns a PostgreSQL SELECT that maps
// employee_id → current_grade name from ACTIVE EmployeeGrade rows.
//
// DISTINCT ON (employee_id) ORDER BY start_date DESC, id DESC is a defensive
// fallback for dirty pre-migration data with multiple ACTIVE rows; under the
// partial unique index there is at most one ACTIVE row per employee.
func buildActiveCurrentGradeSelectSQL() string {
	egTable := domain.GetTableName("hr_employee_grades")
	gradesTable := domain.GetTableName("hr_grades")
	return fmt.Sprintf(`
			SELECT DISTINCT ON (eg.employee_id)
				eg.employee_id,
				eg.grade_id AS current_grade_id,
				g.name AS current_grade
			FROM %s eg
			LEFT JOIN %s g
				ON g.id = eg.grade_id AND g.deleted = false
			WHERE %s
			ORDER BY eg.employee_id, eg.start_date DESC, eg.id DESC
	`, egTable, gradesTable, activeEmployeeGradeSQLConditions("eg"))
}

// applyActiveEmployeeGradeIDFilter restricts employees that have an ACTIVE
// (deleted=false) EmployeeGrade for the given grade_id.
// Uses EXISTS to avoid duplicate rows and keep list/count pagination aligned.
func applyActiveEmployeeGradeIDFilter(query *gorm.DB, employeeTable string, gradeID interface{}) *gorm.DB {
	egTable := domain.GetTableName("hr_employee_grades")
	sql := fmt.Sprintf(`EXISTS (
		SELECT 1 FROM %s eg
		WHERE eg.employee_id = %s.id
		  AND eg.deleted = ?
		  AND eg.status = ?
		  AND eg.grade_id = ?
	)`, egTable, employeeTable)

	return query.Where(sql, false, domain.EmployeeGradeStatusActive, gradeID)
}
