package jobs

import (
	"log"

	"gorm.io/gorm"
)

// LeaveBalanceJob handles the annual leave balance updates
type LeaveBalanceJob struct {
	db *gorm.DB
}

// NewLeaveBalanceJob creates a new leave balance job
func NewLeaveBalanceJob(db *gorm.DB) *LeaveBalanceJob {
	return &LeaveBalanceJob{db: db}
}

// Run executes the leave balance update job
func (j *LeaveBalanceJob) Run() {
	log.Println("[LeaveBalanceJob] Starting annual leave balance update...")

	query := `
WITH calc AS (
    SELECT
        e.id AS employee_id,
        CASE
            WHEN years_of_service BETWEEN 1 AND 5 THEN 14
            WHEN years_of_service BETWEEN 6 AND 14 THEN 20
            WHEN years_of_service >= 15 THEN 26
            ELSE 0
        END AS leave_days
    FROM (
        SELECT
            id,
            hire_date,
            EXTRACT(YEAR FROM AGE(CURRENT_DATE, hire_date))::INTEGER AS years_of_service
        FROM hr_employees
        WHERE deleted = false
          AND (
              (EXTRACT(MONTH FROM hire_date) = EXTRACT(MONTH FROM CURRENT_DATE)
                  AND EXTRACT(DAY FROM hire_date) = EXTRACT(DAY FROM CURRENT_DATE))
              OR
              (EXTRACT(MONTH FROM hire_date) = 2
                  AND EXTRACT(DAY FROM hire_date) = 29
                  AND EXTRACT(MONTH FROM CURRENT_DATE) = 2
                  AND EXTRACT(DAY FROM CURRENT_DATE) = 28
                  AND EXTRACT(YEAR FROM CURRENT_DATE)::INTEGER % 4 != 0)
          )
    ) e
    WHERE (
        CASE
            WHEN years_of_service BETWEEN 1 AND 5 THEN 14
            WHEN years_of_service BETWEEN 6 AND 14 THEN 20
            WHEN years_of_service >= 15 THEN 26
            ELSE 0
        END
    ) > 0
),

updated AS (
    UPDATE hr_leave_balances lb
    SET
        total_days = lb.total_days + calc.leave_days,
        remaining_days = lb.remaining_days + calc.leave_days,
        updated_at = CURRENT_TIMESTAMP
    FROM calc
    WHERE lb.employee_id = calc.employee_id
      AND lb.leave_type_id = 1
      AND lb.deleted = false
    RETURNING lb.employee_id
)

INSERT INTO hr_leave_balances (
    employee_id,
    leave_type_id,
    total_days,
    used_days,
    remaining_days,
    deleted,
    created_at,
    updated_at
)
SELECT
    calc.employee_id,
    1,
    calc.leave_days,
    0,
    calc.leave_days,
    false,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM calc
LEFT JOIN updated u ON u.employee_id = calc.employee_id
WHERE u.employee_id IS NULL;
`

	result := j.db.Exec(query)
	if result.Error != nil {
		log.Printf("[LeaveBalanceJob] Error: %v", result.Error)
		return
	}

	log.Printf("[LeaveBalanceJob] Completed successfully. Rows affected: %d", result.RowsAffected)
}
