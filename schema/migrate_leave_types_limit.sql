-- Migration: Change is_limited boolean to limit_amount integer
-- Date: 2026-03-06

-- Step 1: Add new column limit_amount
ALTER TABLE hr_leave_types ADD COLUMN IF NOT EXISTS limit_amount INTEGER NULL;

-- Step 2: Migrate data - if is_limited was true, we need manual update for specific amounts
-- For now, set NULL for all (unlimited) or specific amounts can be set manually
UPDATE hr_leave_types SET limit_amount = NULL WHERE is_limited = false;

-- Step 3: Drop old column is_limited
ALTER TABLE hr_leave_types DROP COLUMN IF EXISTS is_limited;

-- Note: After running this migration, please update specific leave types with their limit amounts
-- Example: UPDATE hr_leave_types SET limit_amount = 14 WHERE name = 'Annual Leave';
