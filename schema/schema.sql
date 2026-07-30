-- Kartezya HR Management System Database Schema
-- Generated: January 3, 2026
-- Description: Complete database schema for HRMS with enums, tables, and indexes

-- Enable UUID extension for future use
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";



-- ================================================
-- AUTHENTICATION & AUTHORIZATION TABLES
-- ================================================

-- Users table (no is_active field)
CREATE TABLE IF NOT EXISTS hr_users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    password_reset_token VARCHAR(255),
    password_reset_expires TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50)
);

-- Roles table
CREATE TABLE IF NOT EXISTS hr_roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50)
);

-- User-Role junction table
CREATE TABLE IF NOT EXISTS hr_user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (user_id) REFERENCES hr_users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES hr_roles(id) ON DELETE CASCADE,
    UNIQUE(user_id, role_id)
);

-- ================================================
-- COMPANY & ORGANIZATION TABLES
-- ================================================

-- Companies table
CREATE TABLE IF NOT EXISTS hr_companies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    phone VARCHAR(20),
    email VARCHAR(255),
    website VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50)
);

-- Departments table
CREATE TABLE IF NOT EXISTS hr_departments (
    id SERIAL PRIMARY KEY,
    company_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    manager TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (company_id) REFERENCES hr_companies(id) ON DELETE CASCADE
);

-- Job Positions table (no department relationship as per requirements)
CREATE TABLE IF NOT EXISTS hr_job_positions (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50)
);

-- ================================================
-- EMPLOYEE TABLES
-- ================================================

-- Employees table with all new fields
CREATE TABLE IF NOT EXISTS hr_employees (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    address TEXT,
    state VARCHAR(100),
    city VARCHAR(30),
    gender VARCHAR(20),
    date_of_birth DATE,
    hire_date DATE,
    leave_date DATE,
    marital_status VARCHAR(20),
    emergency_contact VARCHAR(15),
    emergency_contact_name VARCHAR(20),
    emergency_contact_relation VARCHAR(20),
    grade_id BIGINT,
    profession_start_date DATE,
    note TEXT,
    father_name VARCHAR(255),
    nationality VARCHAR(100),
    identity_no VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (user_id) REFERENCES hr_users(id) ON DELETE CASCADE,
    FOREIGN KEY (grade_id) REFERENCES hr_grades(id) ON DELETE SET NULL
);

-- Employee Work Information (updated structure: no salary, no is_active, added company_id and department_id)
CREATE TABLE IF NOT EXISTS hr_employee_work_information (
    id SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL,
    company_id INTEGER NOT NULL,
    department_id INTEGER NOT NULL,
    job_position_id INTEGER NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    personnel_no VARCHAR(100),
    work_email VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (employee_id) REFERENCES hr_employees(id) ON DELETE CASCADE,
    FOREIGN KEY (company_id) REFERENCES hr_companies(id) ON DELETE CASCADE,
    FOREIGN KEY (department_id) REFERENCES hr_departments(id) ON DELETE CASCADE,
    FOREIGN KEY (job_position_id) REFERENCES hr_job_positions(id) ON DELETE CASCADE
);

-- ================================================
-- LEAVE MANAGEMENT TABLES
-- ================================================

-- Leave Types table with new boolean fields
CREATE TABLE IF NOT EXISTS hr_leave_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_paid BOOLEAN NOT NULL DEFAULT false,
    limit_amount INTEGER NULL,
    is_accrual BOOLEAN NOT NULL DEFAULT false,
    is_required_document BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50)
);

-- Leave Balances table
CREATE TABLE IF NOT EXISTS hr_leave_balances (
    id SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL,
    leave_type_id INTEGER NOT NULL,
    year INTEGER NOT NULL,
    total_days DOUBLE PRECISION NOT NULL,
    used_days DOUBLE PRECISION DEFAULT 0,
    remaining_days DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (employee_id) REFERENCES hr_employees(id) ON DELETE CASCADE,
    FOREIGN KEY (leave_type_id) REFERENCES hr_leave_types(id) ON DELETE CASCADE,
    UNIQUE(employee_id, leave_type_id, year)
);

-- Leave Requests table (updated to remove leave_sub_type_id)
CREATE TABLE IF NOT EXISTS hr_leave_requests (
    id SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL,
    leave_type_id INTEGER NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_start_date_full_day BOOLEAN NOT NULL DEFAULT true,
    is_finish_date_full_day BOOLEAN NOT NULL DEFAULT true,
    requested_days DOUBLE PRECISION NOT NULL,
    reason TEXT,
    status VARCHAR(20) DEFAULT 'PENDING',
    is_paid BOOLEAN NOT NULL DEFAULT false,
    approved_by INTEGER,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (employee_id) REFERENCES hr_employees(id) ON DELETE CASCADE,
    FOREIGN KEY (leave_type_id) REFERENCES hr_leave_types(id) ON DELETE CASCADE,
    FOREIGN KEY (approved_by) REFERENCES hr_users(id)
);

-- Leave Documents table
CREATE TABLE IF NOT EXISTS hr_leave_documents (
    id SERIAL PRIMARY KEY,
    leave_request_id INTEGER NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    mime_type VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (leave_request_id) REFERENCES hr_leave_requests(id) ON DELETE CASCADE
);

-- ================================================
-- AUDIT LOG TABLE
-- ================================================

-- Audit Log table (no soft delete - append only)
CREATE TABLE IF NOT EXISTS hr_audit_logs (
    id SERIAL PRIMARY KEY,
    entity_name VARCHAR(100) NOT NULL,
    entity_id INTEGER NOT NULL,
    action VARCHAR(20) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    created_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER NOT NULL,
    FOREIGN KEY (created_by) REFERENCES hr_users(id)
);

CREATE TABLE hr_holidays (
                                 id BIGSERIAL PRIMARY KEY,
                                 holiday_date DATE NOT NULL,
                                 name VARCHAR(100) NOT NULL,
                                 is_full_day BOOLEAN NOT NULL,
                                 created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                 updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                 deleted BOOLEAN NOT NULL DEFAULT false,
                                 created_by VARCHAR(50) DEFAULT 'admin@kartezya.com',
                                 modified_by VARCHAR(50) DEFAULT 'admin@kartezya.com'
);

-- Grades table
CREATE TABLE IF NOT EXISTS hr_grades (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    min_year INTEGER,
    max_year INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50)
);

-- Employee Grades table (grade history/progression for employees)
CREATE TABLE IF NOT EXISTS hr_employee_grades (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL,
    grade_id BIGINT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (employee_id) REFERENCES hr_employees(id) ON DELETE CASCADE,
    FOREIGN KEY (grade_id) REFERENCES hr_grades(id) ON DELETE CASCADE
);

-- Employee Contracts table (contract history for employees)
CREATE TABLE IF NOT EXISTS hr_employee_contracts (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL,
    contract_no VARCHAR(100),
    start_date DATE NOT NULL,
    end_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (employee_id) REFERENCES hr_employees(id) ON DELETE CASCADE
);

-- ================================================
-- INDEXES FOR PERFORMANCE
-- ================================================

-- Authentication indexes
CREATE INDEX IF NOT EXISTS idx_hr_users_email ON hr_users(email);
CREATE INDEX IF NOT EXISTS idx_hr_users_deleted ON hr_users(deleted);

-- Employee indexes
CREATE INDEX IF NOT EXISTS idx_hr_employees_user_id ON hr_employees(user_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_deleted ON hr_employees(deleted);
CREATE INDEX IF NOT EXISTS idx_hr_employees_hire_date ON hr_employees(hire_date);
CREATE INDEX IF NOT EXISTS idx_hr_employees_leave_date ON hr_employees(leave_date);
CREATE INDEX IF NOT EXISTS idx_hr_employees_gender ON hr_employees(gender);

-- Leave management indexes
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_employee_id ON hr_leave_requests(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_status ON hr_leave_requests(status);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_dates ON hr_leave_requests(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_type ON hr_leave_requests(leave_type_id);
CREATE INDEX IF NOT EXISTS idx_hr_leave_balances_employee_year ON hr_leave_balances(employee_id, year);

-- Employee grades indexes
CREATE INDEX IF NOT EXISTS idx_hr_employee_grades_employee_id ON hr_employee_grades(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_employee_grades_grade_id ON hr_employee_grades(grade_id);
CREATE INDEX IF NOT EXISTS idx_hr_employee_grades_dates ON hr_employee_grades(start_date, end_date);

-- Employee contracts indexes
CREATE INDEX IF NOT EXISTS idx_hr_employee_contracts_employee_id ON hr_employee_contracts(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_employee_contracts_dates ON hr_employee_contracts(start_date, end_date);

-- Audit indexes
CREATE INDEX IF NOT EXISTS idx_hr_audit_logs_entity ON hr_audit_logs(entity_name, entity_id);
CREATE INDEX IF NOT EXISTS idx_hr_audit_logs_created_date ON hr_audit_logs(created_date);

-- ================================================
-- TRIGGERS FOR UPDATED_AT TIMESTAMPS
-- ================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

-- Add triggers to all tables with updated_at column
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_users_updated_at') THEN
        CREATE TRIGGER update_hr_users_updated_at BEFORE UPDATE ON hr_users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_roles_updated_at') THEN
        CREATE TRIGGER update_hr_roles_updated_at BEFORE UPDATE ON hr_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_user_roles_updated_at') THEN
        CREATE TRIGGER update_hr_user_roles_updated_at BEFORE UPDATE ON hr_user_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_companies_updated_at') THEN
        CREATE TRIGGER update_hr_companies_updated_at BEFORE UPDATE ON hr_companies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_departments_updated_at') THEN
        CREATE TRIGGER update_hr_departments_updated_at BEFORE UPDATE ON hr_departments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_job_positions_updated_at') THEN
        CREATE TRIGGER update_hr_job_positions_updated_at BEFORE UPDATE ON hr_job_positions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_employees_updated_at') THEN
        CREATE TRIGGER update_hr_employees_updated_at BEFORE UPDATE ON hr_employees FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_employee_work_information_updated_at') THEN
        CREATE TRIGGER update_hr_employee_work_information_updated_at BEFORE UPDATE ON hr_employee_work_information FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_leave_types_updated_at') THEN
        CREATE TRIGGER update_hr_leave_types_updated_at BEFORE UPDATE ON hr_leave_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_leave_balances_updated_at') THEN
        CREATE TRIGGER update_hr_leave_balances_updated_at BEFORE UPDATE ON hr_leave_balances FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_leave_requests_updated_at') THEN
        CREATE TRIGGER update_hr_leave_requests_updated_at BEFORE UPDATE ON hr_leave_requests FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_leave_documents_updated_at') THEN
        CREATE TRIGGER update_hr_leave_documents_updated_at BEFORE UPDATE ON hr_leave_documents FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_grades_updated_at') THEN
        CREATE TRIGGER update_hr_grades_updated_at BEFORE UPDATE ON hr_grades FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_employee_grades_updated_at') THEN
        CREATE TRIGGER update_hr_employee_grades_updated_at BEFORE UPDATE ON hr_employee_grades FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_employee_contracts_updated_at') THEN
        CREATE TRIGGER update_hr_employee_contracts_updated_at BEFORE UPDATE ON hr_employee_contracts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_hr_holidays_updated_at') THEN
        CREATE TRIGGER update_hr_holidays_updated_at BEFORE UPDATE ON hr_holidays FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;