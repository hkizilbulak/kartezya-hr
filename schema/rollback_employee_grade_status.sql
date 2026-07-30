-- rollback_employee_grade_status.sql
--
-- DRAFT / MANUAL ONLY — do not run blindly.
-- Use after a successful status migration when application deploy must roll back
-- to a build that still reads employees.grade_id as source of truth.
--
-- PREFIX: logical hr_* names — adjust for environment.
-- Does NOT drop status column by default (safe leave-behind).
-- Does NOT drop employee legacy columns.

BEGIN;

-- A) Optional: refill employees.grade_id from ACTIVE EmployeeGrade
--    Only when rolling back to pre-Phase-4 application code.
UPDATE hr_employees e
SET grade_id = eg.grade_id
FROM hr_employee_grades eg
WHERE eg.employee_id = e.id
  AND eg.deleted = false
  AND eg.status = 'ACTIVE'
  AND eg.end_date IS NULL;

-- Employees with no ACTIVE row keep existing grade_id (may be stale/null).
-- Review:
-- SELECT e.id, e.grade_id, eg.id, eg.grade_id
-- FROM hr_employees e
-- LEFT JOIN hr_employee_grades eg
--   ON eg.employee_id = e.id AND eg.deleted = false AND eg.status = 'ACTIVE';

-- B) Optional: drop constraints/indexes if a full schema rollback is required
-- DROP INDEX IF EXISTS ux_hr_employee_grades_employee_id_status_active;
-- DROP INDEX IF EXISTS idx_hr_employee_grades_grade_id_active_lookup;
-- ALTER TABLE hr_employee_grades DROP CONSTRAINT IF EXISTS chk_hr_employee_grades_status_end_date;
-- ALTER TABLE hr_employee_grades DROP CONSTRAINT IF EXISTS chk_hr_employee_grades_dates_valid;

-- C) status column: prefer KEEP. Dropping loses ACTIVE/INACTIVE labels.
-- ALTER TABLE hr_employee_grades ALTER COLUMN status DROP NOT NULL;
-- -- Do not DROP COLUMN status unless explicitly approved.

COMMIT;
