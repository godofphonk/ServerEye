package storage

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type KeysStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

func NewKeysStorage(databaseURL string, logger *logrus.Logger) (*KeysStorage, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	storage := &KeysStorage{
		db:     db,
		logger: logger,
	}

	// Initialize schema for keys
	if err := storage.initSchema(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *KeysStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS generated_keys (
		id BIGSERIAL PRIMARY KEY,
		secret_key TEXT UNIQUE NOT NULL,
		agent_version TEXT,
		os_info TEXT,
		hostname TEXT,
		status TEXT DEFAULT 'generated',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_generated_keys_secret ON generated_keys (secret_key);
	CREATE INDEX IF NOT EXISTS idx_generated_keys_status ON generated_keys (status);
	CREATE INDEX IF NOT EXISTS idx_generated_keys_created_at ON generated_keys (created_at);

	-- Create trigger for updated_at
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	CREATE TRIGGER update_generated_keys_updated_at BEFORE UPDATE
		ON generated_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *KeysStorage) InsertGeneratedKey(secretKey, agentVersion, osInfo, hostname string) error {
	query := `
		INSERT INTO generated_keys (secret_key, agent_version, os_info, hostname, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (secret_key) DO NOTHING
	`

	_, err := s.db.Exec(query, secretKey, agentVersion, osInfo, hostname)
	return err
}

func (s *KeysStorage) Close() error {
	return s.db.Close()
}
