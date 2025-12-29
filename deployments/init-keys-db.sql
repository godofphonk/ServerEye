-- Initialize database for ServerEye key registration
-- This script runs automatically when the container starts

CREATE TABLE IF NOT EXISTS generated_keys (
    id SERIAL PRIMARY KEY,
    secret_key VARCHAR(64) UNIQUE NOT NULL,
    agent_version VARCHAR(50),
    os_info VARCHAR(100),
    hostname VARCHAR(255),
    status VARCHAR(20) DEFAULT 'generated',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_generated_keys_secret_key ON generated_keys(secret_key);
CREATE INDEX IF NOT EXISTS idx_generated_keys_status ON generated_keys(status);
CREATE INDEX IF NOT EXISTS idx_generated_keys_created_at ON generated_keys(created_at);

-- Create read-only user for key validation (optional)
-- CREATE USER IF NOT EXISTS servereye_keys_readonly WITH PASSWORD 'readonly_password';
-- GRANT SELECT ON generated_keys TO servereye_keys_readonly;

-- Create trigger for updated_at
DROP TRIGGER IF EXISTS update_generated_keys_updated_at ON generated_keys;
DROP FUNCTION IF EXISTS update_updated_at_column();

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_generated_keys_updated_at BEFORE UPDATE
    ON generated_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
