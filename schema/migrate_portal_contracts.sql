-- Create portal contracts table
CREATE TABLE IF NOT EXISTS hr_portal_contracts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    version VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50)
);

-- Create employee portal contracts pivot table
CREATE TABLE IF NOT EXISTS hr_employee_portal_contracts (
    id SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL,
    contract_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approved_at TIMESTAMP WITH TIME ZONE,
    ip_address VARCHAR(45),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    FOREIGN KEY (employee_id) REFERENCES hr_employees(id) ON DELETE CASCADE,
    FOREIGN KEY (contract_id) REFERENCES hr_portal_contracts(id) ON DELETE CASCADE,
    UNIQUE(employee_id, contract_id)
);

-- Create performance indexes
CREATE INDEX IF NOT EXISTS idx_hr_employee_portal_contracts_emp ON hr_employee_portal_contracts(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_employee_portal_contracts_contract ON hr_employee_portal_contracts(contract_id);
