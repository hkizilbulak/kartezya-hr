-- KVKK User Settings Table
CREATE TABLE IF NOT EXISTS hr_test_user_settings (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(50),
    modified_by VARCHAR(50),
    user_id INT NOT NULL UNIQUE,
    kvkk_approved BOOLEAN DEFAULT FALSE,
    kvkk_approved_at TIMESTAMP WITH TIME ZONE,
    kvkk_rejected_at TIMESTAMP WITH TIME ZONE,
    promotion_email_allowed BOOLEAN DEFAULT TRUE,
    promotion_sms_allowed BOOLEAN DEFAULT TRUE
);

-- KVKK Log Table
CREATE TABLE IF NOT EXISTS hr_test_kvkk_logs (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    action VARCHAR(20) NOT NULL,
    client_ip VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Upgrades for "Daha Sonra Hatırlat" (Remind Later)
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS kvkk_status VARCHAR(20) DEFAULT 'PENDING' NOT NULL;
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS kvkk_last_postponed_at TIMESTAMP WITH TIME ZONE;

-- Upgrades for Multiple Documents and Policies
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS photo_consent VARCHAR(20) DEFAULT 'PENDING' NOT NULL;
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS kvkk_text VARCHAR(20) DEFAULT 'PENDING' NOT NULL;
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS privacy_policy VARCHAR(20) DEFAULT 'PENDING' NOT NULL;
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS anti_bribery_policy VARCHAR(20) DEFAULT 'PENDING' NOT NULL;

ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS photo_consent_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS kvkk_text_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS privacy_policy_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE hr_test_user_settings ADD COLUMN IF NOT EXISTS anti_bribery_policy_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE hr_test_kvkk_logs ADD COLUMN IF NOT EXISTS document_type VARCHAR(50) DEFAULT 'ALL' NOT NULL;


