-- migrate_employee_grade_status.sql
--
-- EmployeeGrade ACTIVE/INACTIVE status backfill + constraints (MANUAL / controlled).
--
-- OPERASYON SIRASI (bkz. EMPLOYEE_GRADE_MIGRATION_RUNBOOK.md):
--   1) Backup
--   2) schema/diagnose_employee_grade_migration.sql (read-only) incele
--   3) Bu dosyayı TEK SEFERDE transaction içinde çalıştır
--   4) Doğrulama SELECT sonuçlarını kontrol et
--
-- AutoMigrate/Ensure (DB_AUTO_MIGRATE=true) yalnız status kolonunu ekler;
-- VERİ BACKFILL ve CONSTRAINT bu dosyadadır. Ensure ile bu SQL'i aynı anda
-- "çift backfill" olarak koşturmaayın — Ensure artık backfill yapmaz.
--
-- PREFIX: Logical hr_* adları kullanılır (diğer schema/migrate_*.sql gibi).
-- Ortam prefix'i hr_test / hr_prod ise tablo adlarını (ve chk_/ux_/idx_ isimlerini)
-- ortama göre değiştirin. Go Ensure path prefix-aware'dir; bu SQL mirror'dır.
--
-- employees.grade_id / contract_no / mother_name / total_gap / is_grade_up DROP YOK.
--
-- Aktif seçim (deleted=false):
--   1) status=ACTIVE ve end_date IS NULL
--   2) end_date IS NULL
--   3) CURRENT_DATE aralığında
--   4) start_date DESC, created_at DESC, id DESC
-- Soft-deleted → INACTIVE + end_date doldurulur.

BEGIN;

-- 1) Schema: status kolonu (idempotent)
ALTER TABLE hr_employee_grades ADD COLUMN IF NOT EXISTS status VARCHAR(20);

-- 2) Precheck / guard — kritik anomalide transaction abort
DO $precheck$
BEGIN
	IF EXISTS (
		SELECT 1 FROM hr_employee_grades eg
		LEFT JOIN hr_employees e ON e.id = eg.employee_id
		WHERE e.id IS NULL
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: orphan employee_id on hr_employee_grades';
	END IF;

	IF EXISTS (
		SELECT 1 FROM hr_employee_grades eg
		LEFT JOIN hr_grades g ON g.id = eg.grade_id
		WHERE g.id IS NULL
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: orphan grade_id on hr_employee_grades';
	END IF;

	IF EXISTS (
		SELECT 1 FROM hr_employee_grades WHERE start_date IS NULL
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: NULL start_date on hr_employee_grades';
	END IF;

	IF EXISTS (
		SELECT 1 FROM hr_employee_grades
		WHERE end_date IS NOT NULL AND end_date < start_date
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: end_date < start_date on hr_employee_grades';
	END IF;

	IF EXISTS (
		SELECT 1
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
	) THEN
		RAISE EXCEPTION 'employee_grade_status precheck BLOCKER: overlapping closed date ranges — run diagnose_employee_grade_migration.sql';
	END IF;
END
$precheck$;

-- 3) Deterministic backfill (auto-fix: multiple open ends, missing status)
WITH ranked AS (
	SELECT
		id,
		ROW_NUMBER() OVER (
			PARTITION BY employee_id
			ORDER BY
				CASE WHEN status = 'ACTIVE' AND end_date IS NULL THEN 0 ELSE 1 END,
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

-- 4) Lock status column
ALTER TABLE hr_employee_grades ALTER COLUMN status SET DEFAULT 'ACTIVE';
UPDATE hr_employee_grades SET status = 'ACTIVE' WHERE status IS NULL;
ALTER TABLE hr_employee_grades ALTER COLUMN status SET NOT NULL;

-- 5) Post-backfill assert
DO $assert$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM hr_employee_grades
		WHERE status = 'ACTIVE' AND deleted = false
		GROUP BY employee_id
		HAVING COUNT(*) > 1
	) THEN
		RAISE EXCEPTION 'employee_grade_status assert: multiple ACTIVE grades remain on hr_employee_grades';
	END IF;

	IF EXISTS (
		SELECT 1 FROM hr_employee_grades
		WHERE (end_date IS NOT NULL AND end_date < start_date)
		   OR (status = 'ACTIVE' AND (end_date IS NOT NULL OR deleted = true))
		   OR (status = 'INACTIVE' AND end_date IS NULL)
		   OR (status IS NULL OR status NOT IN ('ACTIVE', 'INACTIVE'))
	) THEN
		RAISE EXCEPTION 'employee_grade_status assert: status/end_date invariants still violated on hr_employee_grades';
	END IF;
END
$assert$;

-- 6) CHECK constraints (idempotent drop/add)
ALTER TABLE hr_employee_grades DROP CONSTRAINT IF EXISTS chk_hr_employee_grades_dates_valid;
ALTER TABLE hr_employee_grades ADD CONSTRAINT chk_hr_employee_grades_dates_valid
	CHECK (end_date IS NULL OR end_date >= start_date);

ALTER TABLE hr_employee_grades DROP CONSTRAINT IF EXISTS chk_hr_employee_grades_status_end_date;
ALTER TABLE hr_employee_grades ADD CONSTRAINT chk_hr_employee_grades_status_end_date
	CHECK (
		(status = 'ACTIVE' AND end_date IS NULL AND deleted = false)
		OR (status = 'INACTIVE' AND end_date IS NOT NULL)
	);

-- 7) Indexes (idempotent)
CREATE UNIQUE INDEX IF NOT EXISTS ux_hr_employee_grades_employee_id_status_active
	ON hr_employee_grades (employee_id)
	WHERE status = 'ACTIVE' AND deleted = false;

CREATE INDEX IF NOT EXISTS idx_hr_employee_grades_grade_id_active_lookup
	ON hr_employee_grades (grade_id, employee_id)
	WHERE status = 'ACTIVE' AND deleted = false;

-- 8) Read-only verification (does not mutate)
SELECT 'status_distribution' AS check_name, status, COUNT(*) AS cnt
FROM hr_employee_grades
GROUP BY status
ORDER BY status;

SELECT 'multi_active_employees' AS check_name, COUNT(*) AS cnt
FROM (
	SELECT employee_id
	FROM hr_employee_grades
	WHERE status = 'ACTIVE' AND deleted = false
	GROUP BY employee_id
	HAVING COUNT(*) > 1
) x;

SELECT 'invariant_violations' AS check_name, COUNT(*) AS cnt
FROM hr_employee_grades
WHERE (end_date IS NOT NULL AND end_date < start_date)
   OR (status = 'ACTIVE' AND (end_date IS NOT NULL OR deleted = true))
   OR (status = 'INACTIVE' AND end_date IS NULL);

COMMIT;
