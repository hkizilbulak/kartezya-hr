-- migrate_employee_grade_status.sql
--
-- EmployeeGrade ACTIVE/INACTIVE status model + backfill + constraints.
--
-- SOURCE OF TRUTH: Prefer EnsureEmployeeGradeStatusConstraints in
-- internal/database/employee_grade_status.go (wired from Database.Migrate when
-- DB_AUTO_MIGRATE=true). That path is prefix-aware via domain.GetTableName.
--
-- Run THIS file only when AutoMigrate/Ensure is disabled, and only once.
-- Do NOT run this SQL and Ensure on the same database (duplicate / conflict risk).
--
-- Uses logical hr_* names (same convention as other schema/migrate_*.sql files).
-- Does NOT touch hr_employees.grade_id (later phase).
--
-- Active row selection (per employee, deleted=false):
--   1) end_date IS NULL
--   2) CURRENT_DATE in [start_date, end_date]
--   3) start_date DESC, created_at DESC, id DESC
-- Winner → status=ACTIVE, end_date=NULL
-- Others → status=INACTIVE; null end_date closed as next_start-1 day (or start_date)
-- Soft-deleted rows → INACTIVE with end_date filled; excluded from unique index.

ALTER TABLE hr_employee_grades ADD COLUMN IF NOT EXISTS status VARCHAR(20);

WITH ranked AS (
	SELECT
		id,
		ROW_NUMBER() OVER (
			PARTITION BY employee_id
			ORDER BY
				CASE WHEN end_date IS NULL THEN 0 ELSE 1 END,
				CASE
					WHEN start_date <= CURRENT_DATE
						AND (end_date IS NULL OR end_date >= CURRENT_DATE)
					THEN 0 ELSE 1
				END,
				start_date DESC,
				created_at DESC,
				id DESC
		) AS rn
	FROM hr_employee_grades
	WHERE deleted = false
),
with_next AS (
	SELECT
		id,
		start_date,
		LEAD(start_date) OVER (
			PARTITION BY employee_id
			ORDER BY start_date ASC, id ASC
		) AS next_start
	FROM hr_employee_grades
	WHERE deleted = false
),
prepared AS (
	SELECT
		eg.id,
		CASE
			WHEN eg.deleted = true THEN 'INACTIVE'
			WHEN r.rn = 1 THEN 'ACTIVE'
			ELSE 'INACTIVE'
		END AS new_status,
		CASE
			WHEN eg.deleted = true THEN COALESCE(eg.end_date, eg.start_date)
			WHEN r.rn = 1 THEN NULL::date
			WHEN eg.end_date IS NOT NULL THEN eg.end_date
			WHEN n.next_start IS NOT NULL
				AND (n.next_start - INTERVAL '1 day')::date >= eg.start_date
				THEN (n.next_start - INTERVAL '1 day')::date
			ELSE eg.start_date
		END AS new_end_date
	FROM hr_employee_grades eg
	LEFT JOIN ranked r ON r.id = eg.id
	LEFT JOIN with_next n ON n.id = eg.id
)
UPDATE hr_employee_grades eg
SET
	status = p.new_status,
	end_date = p.new_end_date
FROM prepared p
WHERE eg.id = p.id;

ALTER TABLE hr_employee_grades ALTER COLUMN status SET DEFAULT 'ACTIVE';
UPDATE hr_employee_grades SET status = 'ACTIVE' WHERE status IS NULL;
ALTER TABLE hr_employee_grades ALTER COLUMN status SET NOT NULL;

-- Fail the session if invariants are still broken (ops must inspect before continuing).
DO $$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM hr_employee_grades
		WHERE status = 'ACTIVE' AND deleted = false
		GROUP BY employee_id
		HAVING COUNT(*) > 1
	) THEN
		RAISE EXCEPTION 'migrate_employee_grade_status: multiple ACTIVE grades remain for at least one employee';
	END IF;

	IF EXISTS (
		SELECT 1 FROM hr_employee_grades
		WHERE (end_date IS NOT NULL AND end_date < start_date)
		   OR (status = 'ACTIVE' AND (end_date IS NOT NULL OR deleted = true))
		   OR (status = 'INACTIVE' AND end_date IS NULL)
	) THEN
		RAISE EXCEPTION 'migrate_employee_grade_status: status/end_date invariants still violated';
	END IF;
END $$;

ALTER TABLE hr_employee_grades DROP CONSTRAINT IF EXISTS chk_hr_employee_grades_dates_valid;
ALTER TABLE hr_employee_grades ADD CONSTRAINT chk_hr_employee_grades_dates_valid
	CHECK (end_date IS NULL OR end_date >= start_date);

ALTER TABLE hr_employee_grades DROP CONSTRAINT IF EXISTS chk_hr_employee_grades_status_end_date;
ALTER TABLE hr_employee_grades ADD CONSTRAINT chk_hr_employee_grades_status_end_date
	CHECK (
		(status = 'ACTIVE' AND end_date IS NULL AND deleted = false)
		OR (status = 'INACTIVE' AND end_date IS NOT NULL)
	);

CREATE UNIQUE INDEX IF NOT EXISTS ux_hr_employee_grades_employee_id_status_active
	ON hr_employee_grades (employee_id)
	WHERE status = 'ACTIVE' AND deleted = false;

-- Supports Employees grade_id filter EXISTS (grade_id, employee_id) on ACTIVE rows.
CREATE INDEX IF NOT EXISTS idx_hr_employee_grades_grade_id_active_lookup
	ON hr_employee_grades (grade_id, employee_id)
	WHERE status = 'ACTIVE' AND deleted = false;
