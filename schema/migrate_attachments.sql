-- ==================== Document Management System (DYS) ====================
-- Migration for Generic Attachment/Document Management System
-- Created: 2026-04-03
-- Purpose: Centralized document storage for all modules (Expense, Leave, User, Employee, Contract)

-- ==================== hr_ prefix (Production) ====================

-- Create attachments table
CREATE TABLE IF NOT EXISTS hr_attachments (
    id VARCHAR(36) PRIMARY KEY,                                -- UUID
    owner_id INTEGER NOT NULL,                                 -- User who uploaded the file
    related_type INTEGER NOT NULL,                             -- Module type (1:Expense, 2:Leave, 3:User, 4:Employee, 5:Contract)
    related_id INTEGER,                                        -- Related record ID (nullable until linked)
    type INTEGER NOT NULL,                                     -- Document category (1:Invoice, 2:MedicalReport, 3:Avatar, etc.)
    status INTEGER NOT NULL DEFAULT 1,                         -- Lifecycle status (1:Temporary, 2:Linked, 3:Archived)
    file_name VARCHAR(255) NOT NULL,                           -- Original filename
    path VARCHAR(500) NOT NULL,                                -- Storage path
    content_type VARCHAR(100) NOT NULL,                        -- MIME type
    file_size BIGINT NOT NULL,                                 -- File size in bytes
    hash VARCHAR(64),                                          -- SHA256 hash for duplicate detection
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key constraint
    CONSTRAINT fk_attachments_owner FOREIGN KEY (owner_id) REFERENCES hr_users(id) ON DELETE CASCADE,
    
    -- Indexes for performance
    INDEX idx_attachments_owner (owner_id),
    INDEX idx_attachments_related (related_type, related_id),
    INDEX idx_attachments_status (status),
    INDEX idx_attachments_hash (hash)
);

-- Add comment to table
COMMENT ON TABLE hr_attachments IS 'Generic document/attachment storage for all modules - Single Source of Truth for DYS';

-- Add comments to columns
COMMENT ON COLUMN hr_attachments.id IS 'UUID primary key';
COMMENT ON COLUMN hr_attachments.owner_id IS 'User who uploaded the file';
COMMENT ON COLUMN hr_attachments.related_type IS 'Module type: 1=Expense, 2=Leave, 3=User, 4=Employee, 5=Contract';
COMMENT ON COLUMN hr_attachments.related_id IS 'Related record ID (null until linked)';
COMMENT ON COLUMN hr_attachments.type IS 'Document category: 1=Invoice, 2=MedicalReport, 3=Avatar, 4=Receipt, 5=Contract, 6=Identity, 7=Diploma, 8=Certificate, 99=Other';
COMMENT ON COLUMN hr_attachments.status IS 'Lifecycle status: 1=Temporary (uploaded but not linked), 2=Linked (attached to record), 3=Archived (soft deleted)';
COMMENT ON COLUMN hr_attachments.hash IS 'SHA256 hash for duplicate file detection';

-- Create a function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_hr_attachments_updated_at 
    BEFORE UPDATE ON hr_attachments 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- ==================== hr_test_ prefix (Test/Development) ====================

-- Create attachments table
CREATE TABLE IF NOT EXISTS hr_test_attachments (
    id VARCHAR(36) PRIMARY KEY,                                -- UUID
    owner_id INTEGER NOT NULL,                                 -- User who uploaded the file
    related_type INTEGER NOT NULL,                             -- Module type (1:Expense, 2:Leave, 3:User, 4:Employee, 5:Contract)
    related_id INTEGER,                                        -- Related record ID (nullable until linked)
    type INTEGER NOT NULL,                                     -- Document category (1:Invoice, 2:MedicalReport, 3:Avatar, etc.)
    status INTEGER NOT NULL DEFAULT 1,                         -- Lifecycle status (1:Temporary, 2:Linked, 3:Archived)
    file_name VARCHAR(255) NOT NULL,                           -- Original filename
    path VARCHAR(500) NOT NULL,                                -- Storage path
    content_type VARCHAR(100) NOT NULL,                        -- MIME type
    file_size BIGINT NOT NULL,                                 -- File size in bytes
    hash VARCHAR(64),                                          -- SHA256 hash for duplicate detection
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key constraint
    CONSTRAINT fk_test_attachments_owner FOREIGN KEY (owner_id) REFERENCES hr_test_users(id) ON DELETE CASCADE,
    
    -- Indexes for performance
    INDEX idx_test_attachments_owner (owner_id),
    INDEX idx_test_attachments_related (related_type, related_id),
    INDEX idx_test_attachments_status (status),
    INDEX idx_test_attachments_hash (hash)
);

-- Add comment to table
COMMENT ON TABLE hr_test_attachments IS 'Generic document/attachment storage for all modules - Single Source of Truth for DYS (Test Environment)';

-- Add comments to columns
COMMENT ON COLUMN hr_test_attachments.id IS 'UUID primary key';
COMMENT ON COLUMN hr_test_attachments.owner_id IS 'User who uploaded the file';
COMMENT ON COLUMN hr_test_attachments.related_type IS 'Module type: 1=Expense, 2=Leave, 3=User, 4=Employee, 5=Contract';
COMMENT ON COLUMN hr_test_attachments.related_id IS 'Related record ID (null until linked)';
COMMENT ON COLUMN hr_test_attachments.type IS 'Document category: 1=Invoice, 2=MedicalReport, 3=Avatar, 4=Receipt, 5=Contract, 6=Identity, 7=Diploma, 8=Certificate, 99=Other';
COMMENT ON COLUMN hr_test_attachments.status IS 'Lifecycle status: 1=Temporary (uploaded but not linked), 2=Linked (attached to record), 3=Archived (soft deleted)';
COMMENT ON COLUMN hr_test_attachments.hash IS 'SHA256 hash for duplicate file detection';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_hr_test_attachments_updated_at 
    BEFORE UPDATE ON hr_test_attachments 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

