-- diagnose_employee_grade_migration.sql
--
-- READ-ONLY diagnostic for Employee Grade status migration readiness.
-- ONLY SELECT / read-only CTE. No INSERT/UPDATE/DELETE/ALTER/DROP.
--
-- PREFIX: Logical hr_* table names (same convention as other schema/*.sql).
-- If the environment uses another prefix (e.g. hr_test), rewrite table names
-- before running. Do NOT hardcode a single environment prefix in this file.
--
-- Do NOT connect this file to application AutoMigrate. Run manually in a
-- read-only or DBA session, review results, then decide on migrate_employee_grade_status.sql.
--
-- Status column may or may not exist yet; queries that reference eg.status use
-- information_schema guards or tolerate missing column via dynamic notes.

-- =============================================================================
-- 0) Physical employee column presence (drop readiness later)
-- =============================================================================
-- Adjust table_name if your prefix is not hr_ (e.g. hr_test_employees).
SELECT
	'0_hr_employees_column_presence' AS diagnostic,
	c.column_name,
	c.data_type,
	c.is_nullable
FROM information_schema.columns c
WHERE c.table_schema = current_schema()
  AND c.table_name = 'hr_employees'
  AND c.column_name IN (
	'grade_id', 'is_grade_up', 'contract_no', 'mother_name', 'total_gap', 'total_experience'
  )
ORDER BY c.column_name;

SELECT
	'0b_hr_employee_grades_status_column' AS diagnostic,
	EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'hr_employee_grades'
		  AND column_name = 'status'
	) AS status_column_exists;

-- NOTE: diagnostics 15–20 cast eg.status. If 0b.status_column_exists is false,
-- skip those result sets (they will error until ADD COLUMN / Ensure runs).

-- =============================================================================
-- 1) Totals
-- =============================================================================
SELECT '1_employee_total' AS diagnostic, COUNT(*) AS cnt
FROM hr_employees;

SELECT '2_employee_grade_not_deleted_total' AS diagnostic, COUNT(*) AS cnt
FROM hr_employee_grades
WHERE deleted = false;

-- =============================================================================
-- 3–7) employees.grade_id vs history coverage
-- =============================================================================
SELECT '3_employees_with_grade_id' AS diagnostic, COUNT(*) AS cnt
FROM hr_employees e
WHERE e.deleted = false AND e.grade_id IS NOT NULL;

SELECT
	'4_grade_id_set_but_no_history' AS diagnostic,
	e.id AS employee_id,
	e.first_name,
	e.last_name,
	e.grade_id AS employee_grade_id
FROM hr_employees e
WHERE e.deleted = false
  AND e.grade_id IS NOT NULL
  AND NOT EXISTS (
	SELECT 1 FROM hr_employee_grades eg
	WHERE eg.employee_id = e.id AND eg.deleted = false
  )
ORDER BY e.id;

-- Deterministic "current" history grade (pre-status or with status): same ORDER BY as migration
WITH ranked AS (
	SELECT
		eg.employee_id,
		eg.id AS employee_grade_id,
		eg.grade_id,
		eg.start_date,
		eg.end_date,
		eg.deleted,
		eg.created_at,
		ROW_NUMBER() OVER (
			PARTITION BY eg.employee_id
			ORDER BY
				CASE WHEN eg.end_date IS NULL THEN 0 ELSE 1 END,
				CASE
					WHEN eg.start_date <= CURRENT_DATE
						AND (eg.end_date IS NULL OR eg.end_date >= CURRENT_DATE)
					THEN 0 ELSE 1
				END,
				eg.start_date DESC,
				eg.created_at DESC,
				eg.id DESC
		) AS rn
	FROM hr_employee_grades eg
	WHERE eg.deleted = false
)
SELECT
	'5_employee_grade_id_differs_from_history_current' AS diagnostic,
	e.id AS employee_id,
	e.first_name,
	e.last_name,
	e.grade_id AS employee_grade_id,
	r.employee_grade_id,
	r.grade_id AS history_grade_id,
	r.start_date,
	r.end_date
FROM hr_employees e
JOIN ranked r ON r.employee_id = e.id AND r.rn = 1
WHERE e.deleted = false
  AND e.grade_id IS NOT NULL
  AND e.grade_id <> r.grade_id
ORDER BY e.id;

SELECT
	'6_employees_with_no_employee_grade' AS diagnostic,
	e.id AS employee_id,
	e.first_name,
	e.last_name,
	e.grade_id AS employee_grade_id
FROM hr_employees e
WHERE e.deleted = false
  AND NOT EXISTS (
	SELECT 1 FROM hr_employee_grades eg WHERE eg.employee_id = e.id AND eg.deleted = false
  )
ORDER BY e.id;

SELECT
	'7_employees_only_soft_deleted_history' AS diagnostic,
	e.id AS employee_id,
	e.first_name,
	e.last_name,
	e.grade_id AS employee_grade_id,
	COUNT(eg.id) AS soft_deleted_rows
FROM hr_employees e
JOIN hr_employee_grades eg ON eg.employee_id = e.id AND eg.deleted = true
WHERE e.deleted = false
  AND NOT EXISTS (
	SELECT 1 FROM hr_employee_grades x WHERE x.employee_id = e.id AND x.deleted = false
  )
GROUP BY e.id, e.first_name, e.last_name, e.grade_id
ORDER BY e.id;

-- =============================================================================
-- 8–12) Date anomalies
-- =============================================================================
SELECT
	'8_multiple_open_end_date_null' AS diagnostic,
	eg.employee_id,
	COUNT(*) AS open_rows,
	array_agg(eg.id ORDER BY eg.id) AS employee_grade_ids
FROM hr_employee_grades eg
WHERE eg.deleted = false AND eg.end_date IS NULL
GROUP BY eg.employee_id
HAVING COUNT(*) > 1
ORDER BY eg.employee_id;

SELECT
	'9_multiple_in_current_date_window' AS diagnostic,
	eg.employee_id,
	COUNT(*) AS in_range_rows,
	array_agg(eg.id ORDER BY eg.id) AS employee_grade_ids
FROM hr_employee_grades eg
WHERE eg.deleted = false
  AND eg.start_date <= CURRENT_DATE
  AND (eg.end_date IS NULL OR eg.end_date >= CURRENT_DATE)
GROUP BY eg.employee_id
HAVING COUNT(*) > 1
ORDER BY eg.employee_id;

SELECT
	'10_overlapping_closed_ranges' AS diagnostic,
	a.employee_id,
	a.id AS employee_grade_id_a,
	a.grade_id AS grade_id_a,
	a.start_date AS start_a,
	a.end_date AS end_a,
	b.id AS employee_grade_id_b,
	b.grade_id AS grade_id_b,
	b.start_date AS start_b,
	b.end_date AS end_b
FROM hr_employee_grades a
JOIN hr_employee_grades b
	ON a.employee_id = b.employee_id
	AND a.id < b.id
	AND a.deleted = false
	AND b.deleted = false
	AND a.end_date IS NOT NULL
	AND b.end_date IS NOT NULL
	AND a.start_date <= b.end_date
	AND b.start_date <= a.end_date
ORDER BY a.employee_id, a.id, b.id;

SELECT
	'11_end_before_start' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.start_date,
	eg.end_date,
	eg.deleted
FROM hr_employee_grades eg
WHERE eg.end_date IS NOT NULL AND eg.end_date < eg.start_date
ORDER BY eg.id;

SELECT
	'12_null_start_date' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.end_date,
	eg.deleted
FROM hr_employee_grades eg
WHERE eg.start_date IS NULL
ORDER BY eg.id;

-- =============================================================================
-- 13–14) Orphans
-- =============================================================================
SELECT
	'13_orphan_employee_id' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.start_date,
	eg.end_date,
	eg.deleted,
	eg.created_at
FROM hr_employee_grades eg
LEFT JOIN hr_employees e ON e.id = eg.employee_id
WHERE e.id IS NULL
ORDER BY eg.id;

SELECT
	'14_orphan_grade_id' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.start_date,
	eg.end_date,
	eg.deleted,
	eg.created_at
FROM hr_employee_grades eg
LEFT JOIN hr_grades g ON g.id = eg.grade_id
WHERE g.id IS NULL
ORDER BY eg.id;

-- =============================================================================
-- 15–20) Status-aware checks (skip harmlessly if status missing — run 0c first)
-- =============================================================================
-- Soft-deleted marked ACTIVE (if status exists)
SELECT
	'15_soft_deleted_status_active' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.start_date,
	eg.end_date,
	eg.status::text AS status,
	eg.deleted,
	eg.created_at
FROM hr_employee_grades eg
WHERE eg.deleted = true
  AND eg.status::text = 'ACTIVE'
ORDER BY eg.id;

SELECT
	'16_invalid_or_null_status' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.start_date,
	eg.end_date,
	eg.status::text AS status,
	eg.deleted
FROM hr_employee_grades eg
WHERE eg.status IS NULL
   OR eg.status::text NOT IN ('ACTIVE', 'INACTIVE')
ORDER BY eg.id;

SELECT
	'17_active_with_end_date' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.start_date,
	eg.end_date,
	eg.status::text AS status,
	eg.deleted
FROM hr_employee_grades eg
WHERE eg.status::text = 'ACTIVE' AND eg.end_date IS NOT NULL
ORDER BY eg.id;

SELECT
	'18_inactive_with_null_end_date' AS diagnostic,
	eg.id AS employee_grade_id,
	eg.employee_id,
	eg.grade_id,
	eg.start_date,
	eg.end_date,
	eg.status::text AS status,
	eg.deleted
FROM hr_employee_grades eg
WHERE eg.status::text = 'INACTIVE' AND eg.end_date IS NULL
ORDER BY eg.id;

SELECT
	'19_multiple_active_per_employee' AS diagnostic,
	eg.employee_id,
	COUNT(*) AS active_cnt,
	array_agg(eg.id ORDER BY eg.id) AS employee_grade_ids
FROM hr_employee_grades eg
WHERE eg.status::text = 'ACTIVE' AND eg.deleted = false
GROUP BY eg.employee_id
HAVING COUNT(*) > 1
ORDER BY eg.employee_id;

SELECT
	'20_history_but_no_active' AS diagnostic,
	e.id AS employee_id,
	e.first_name,
	e.last_name,
	COUNT(eg.id) AS history_rows
FROM hr_employees e
JOIN hr_employee_grades eg ON eg.employee_id = e.id AND eg.deleted = false
WHERE e.deleted = false
  AND NOT EXISTS (
	SELECT 1 FROM hr_employee_grades a
	WHERE a.employee_id = e.id AND a.deleted = false AND a.status::text = 'ACTIVE'
  )
GROUP BY e.id, e.first_name, e.last_name
ORDER BY e.id;

-- =============================================================================
-- Decision helper summary (counts only)
-- =============================================================================
SELECT 'summary_orphan_employee' AS diagnostic, COUNT(*) AS cnt FROM (
	SELECT eg.id FROM hr_employee_grades eg
	LEFT JOIN hr_employees e ON e.id = eg.employee_id WHERE e.id IS NULL
) s;

SELECT 'summary_orphan_grade' AS diagnostic, COUNT(*) AS cnt FROM (
	SELECT eg.id FROM hr_employee_grades eg
	LEFT JOIN hr_grades g ON g.id = eg.grade_id WHERE g.id IS NULL
) s;

SELECT 'summary_end_before_start' AS diagnostic, COUNT(*) AS cnt
FROM hr_employee_grades WHERE end_date IS NOT NULL AND end_date < start_date;

SELECT 'summary_closed_overlaps' AS diagnostic, COUNT(*) AS cnt FROM (
	SELECT a.id
	FROM hr_employee_grades a
	JOIN hr_employee_grades b
		ON a.employee_id = b.employee_id AND a.id < b.id
		AND a.deleted = false AND b.deleted = false
		AND a.end_date IS NOT NULL AND b.end_date IS NOT NULL
		AND a.start_date <= b.end_date AND b.start_date <= a.end_date
) s;

SELECT 'summary_multi_open_ends' AS diagnostic, COUNT(*) AS cnt FROM (
	SELECT employee_id FROM hr_employee_grades
	WHERE deleted = false AND end_date IS NULL
	GROUP BY employee_id HAVING COUNT(*) > 1
) s;
