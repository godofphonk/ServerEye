package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/godofphonk/ServerEye/pkg/publisher"
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
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &PostgresStorage{
		db:     db,
		logger: logger,
	}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

func (s *PostgresStorage) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create essential tables only
	schema := `
	-- Create generated_keys table for key registration
	CREATE TABLE IF NOT EXISTS generated_keys (
		id BIGSERIAL PRIMARY KEY,
		secret_key TEXT UNIQUE NOT NULL,
		agent_version TEXT,
		os_info TEXT,
		hostname TEXT,
		status TEXT DEFAULT 'generated',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_generated_keys_secret ON generated_keys (secret_key);

	-- Create servers table for metadata
	CREATE TABLE IF NOT EXISTS servers (
		server_id TEXT PRIMARY KEY,
		server_key TEXT NOT NULL,
		name TEXT,
		last_seen TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_servers_last_seen ON servers (last_seen);

	-- Create dead letter queue for failed messages
	CREATE TABLE IF NOT EXISTS dead_letter_queue (
		id BIGSERIAL PRIMARY KEY,
		topic TEXT NOT NULL,
		partition INTEGER,
		"offset" BIGINT,
		message JSONB NOT NULL,
		error TEXT NOT NULL,
		attempts INTEGER DEFAULT 0,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_dlq_created_at ON dead_letter_queue (created_at);
	CREATE INDEX IF NOT EXISTS idx_dlq_topic ON dead_letter_queue (topic);
	`

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

func (s *PostgresStorage) InsertGeneratedKey(ctx context.Context, secretKey, agentVersion, osInfo, hostname string) error {
	query := `
		INSERT INTO generated_keys (secret_key, agent_version, os_info, hostname, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (secret_key) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, secretKey, agentVersion, osInfo, hostname)
	if err != nil {
		return fmt.Errorf("failed to insert generated key: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"secret_key":    secretKey,
		"agent_version": agentVersion,
		"os_info":       osInfo,
		"hostname":      hostname,
	}).Info("Generated key inserted successfully")

	return nil
}

// Stub implementations for other methods
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

func (s *PostgresStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}
