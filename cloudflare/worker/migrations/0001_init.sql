-- Generated keys table for ServerEye registration
CREATE TABLE IF NOT EXISTS generated_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_key TEXT UNIQUE NOT NULL,
    agent_version TEXT DEFAULT 'unknown',
    os_info TEXT DEFAULT 'unknown',
    hostname TEXT DEFAULT 'unknown',
    status TEXT DEFAULT 'generated',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster lookups by secret_key
CREATE INDEX IF NOT EXISTS idx_generated_keys_secret ON generated_keys(secret_key);

-- Index for time-based queries
CREATE INDEX IF NOT EXISTS idx_generated_keys_created_at ON generated_keys(created_at);

-- Trigger to auto-update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_generated_keys_timestamp
    AFTER UPDATE ON generated_keys
    FOR EACH ROW
BEGIN
    UPDATE generated_keys SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
