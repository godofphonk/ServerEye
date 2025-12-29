package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/godofphonk/ServerEye/pkg/publisher"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type Storage interface {
	StoreMetric(ctx context.Context, metric *publisher.Metric) error
	GetLatestMetrics(ctx context.Context, serverID string) ([]*publisher.Metric, error)
	GetMetricsHistory(ctx context.Context, serverID string, metricType string, from, to time.Time) ([]*publisher.Metric, error)
	GetServers(ctx context.Context) ([]string, error)
	StoreDLQMessage(ctx context.Context, topic string, partition int, offset int64, message []byte, errorMsg string) error
	InsertGeneratedKey(ctx context.Context, secretKey, agentVersion, osInfo, hostname string) error
	Ping() error
	Close() error
}

type PostgresStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

func New(databaseURL string, logger *logrus.Logger) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStorage{
		db:     db,
		logger: logger,
	}, nil
}

func (s *PostgresStorage) StoreMetric(ctx context.Context, metric *publisher.Metric) error {
	s.logger.Warn("StoreMetric called but not implemented")
	return nil
}

func (s *PostgresStorage) GetLatestMetrics(ctx context.Context, serverID string) ([]*publisher.Metric, error) {
	s.logger.Warn("GetLatestMetrics called but not implemented")
	return []*publisher.Metric{}, nil
}

func (s *PostgresStorage) GetMetricsHistory(ctx context.Context, serverID string, metricType string, from, to time.Time) ([]*publisher.Metric, error) {
	s.logger.Warn("GetMetricsHistory called but not implemented")
	return []*publisher.Metric{}, nil
}

func (s *PostgresStorage) GetServers(ctx context.Context) ([]string, error) {
	s.logger.Warn("GetServers called but not implemented")
	return []string{}, nil
}

func (s *PostgresStorage) StoreDLQMessage(ctx context.Context, topic string, partition int, offset int64, message []byte, errorMsg string) error {
	s.logger.Warn("StoreDLQMessage called but not implemented")
	return nil
}

func (s *PostgresStorage) InsertGeneratedKey(ctx context.Context, secretKey, agentVersion, osInfo, hostname string) error {
	s.logger.Warn("InsertGeneratedKey called but not implemented")
	return nil
}

func (s *PostgresStorage) Ping() error {
	return s.db.Ping()
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
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

type KeysStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

func (s *KeysStorage) initSchema() error {
	schema := `
	DROP TRIGGER IF EXISTS update_generated_keys_updated_at ON generated_keys;
	DROP FUNCTION IF EXISTS update_updated_at_column();
	
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
