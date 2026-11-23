-- ============================================================================
-- Migration: Add Inventory Assignment History Table
-- Target Table: hr_inventory_assignment_histories
-- Description: Tracks the lifecycle and assignment history of inventory items
-- ============================================================================

-- 1. Create table
CREATE TABLE IF NOT EXISTS hr_inventory_assignment_histories (
    id SERIAL PRIMARY KEY,
    inventory_item_id INTEGER NOT NULL,
    employee_id INTEGER,
    status VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    assignment_date TIMESTAMP WITH TIME ZONE,
    return_date TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),

    CONSTRAINT fk_inv_hist_item FOREIGN KEY (inventory_item_id) REFERENCES hr_inventory_items(id) ON DELETE CASCADE,
    CONSTRAINT fk_inv_hist_emp FOREIGN KEY (employee_id) REFERENCES hr_employees(id) ON DELETE SET NULL
);

-- 2. Indexes for fast history lookups and active assignment checks
CREATE INDEX IF NOT EXISTS idx_hr_inv_hist_item_id ON hr_inventory_assignment_histories(inventory_item_id);
CREATE INDEX IF NOT EXISTS idx_hr_inv_hist_emp_id ON hr_inventory_assignment_histories(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_inv_hist_active ON hr_inventory_assignment_histories(inventory_item_id) WHERE return_date IS NULL AND deleted = false;

-- 3. Comments
COMMENT ON TABLE hr_inventory_assignment_histories IS 'Stores the historical assignment and return records of inventory items';
COMMENT ON COLUMN hr_inventory_assignment_histories.action IS 'Action type: ASSIGNED, RETURNED, TRANSFERRED, DAMAGED, DELETED';
COMMENT ON COLUMN hr_inventory_assignment_histories.status IS 'Status at event time: IN_USE, IN_STOCK, DAMAGED, RETURNED';

-- 4. Backfill existing assigned inventory items into history (idempotent)
INSERT INTO hr_inventory_assignment_histories (
    inventory_item_id,
    employee_id,
    status,
    action,
    assignment_date,
    created_at,
    updated_at,
    created_by
)
SELECT
    i.id,
    i.employee_id,
    i.status,
    'ASSIGNED',
    COALESCE(i.assignment_date, i.created_at),
    COALESCE(i.assignment_date, i.created_at),
    CURRENT_TIMESTAMP,
    'SYSTEM_BACKFILL'
FROM hr_inventory_items i
WHERE i.employee_id IS NOT NULL
  AND i.deleted = false
  AND NOT EXISTS (
      SELECT 1
      FROM hr_inventory_assignment_histories h
      WHERE h.inventory_item_id = i.id
        AND h.deleted = false
  );

-- ============================================================================
-- ROLLBACK INSTRUCTIONS (Manual/Emergency only):
--
-- DROP TABLE IF EXISTS hr_inventory_assignment_histories CASCADE;
-- ============================================================================
