package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/servereye/servereye-backend/types"
	"github.com/sirupsen/logrus"
	_ "github.com/lib/pq"
)

type Storage interface {
	StoreMetric(ctx context.Context, metric *types.Metric) error
	GetLatestMetrics(ctx context.Context, serverID string) ([]*types.Metric, error)
	GetMetricsHistory(ctx context.Context, serverID string, metricType string, from, to time.Time) ([]*types.Metric, error)
	GetServers(ctx context.Context) ([]string, error)
	StoreDLQMessage(ctx context.Context, topic string, partition int, offset int64, message []byte, errorMsg string) error
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

	// Create metrics table with TimescaleDB hypertable
	schema := `
	-- Enable TimescaleDB extension
	CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

	-- Create metrics table
	CREATE TABLE IF NOT EXISTS metrics (
		id BIGSERIAL,
		server_id TEXT NOT NULL,
		server_key TEXT NOT NULL,
		metric_type TEXT NOT NULL,
		value DOUBLE PRECISION,
		tags JSONB,
		version TEXT,
		timestamp TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_metrics_server_id ON metrics (server_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics (metric_type);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics (timestamp);

	-- Convert to hypertable if not already
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'metrics'
		) THEN
			PERFORM create_hypertable('metrics', 'timestamp', 
				chunk_time_interval => INTERVAL '1 day',
				if_not_exists => TRUE
			);
		END IF;
	END $$;

	-- Create retention policy (keep data for 30 days)
	SELECT add_retention_policy('metrics', INTERVAL '30 days');

	-- Create continuous aggregates for faster queries
	CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_1h
	WITH (timescaledb.continuous) AS
	SELECT 
		time_bucket('1 hour', timestamp) AS bucket,
		server_id,
		metric_type,
		AVG(value) as avg_value,
		MIN(value) as min_value,
		MAX(value) as max_value,
		COUNT(*) as count
	FROM metrics
	GROUP BY bucket, server_id, metric_type;

	-- Refresh policy for continuous aggregate
	SELECT add_continuous_aggregate_policy('metrics_1h',
		start_offset => INTERVAL '1 hour',
		end_offset => INTERVAL '1 hour',
		schedule_interval => INTERVAL '1 hour',
		if_not_exists => TRUE
	);

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
		offset BIGINT,
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

	s.logger.Info("Database schema initialized")
	return nil
}

func (s *PostgresStorage) StoreMetric(ctx context.Context, metric *types.Metric) error {
	query := `
		INSERT INTO metrics (server_id, server_key, metric_type, value, tags, version, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT DO NOTHING
	`

	var tagsJSON []byte
	if metric.Tags != nil {
		var err error
		tagsJSON, err = json.Marshal(metric.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
	}

	_, err := s.db.ExecContext(ctx, query,
		metric.ServerID,
		metric.ServerKey,
		metric.Type,
		metric.Value,
		tagsJSON,
		metric.Version,
		metric.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}

	// Update server last seen
	s.updateServerLastSeen(ctx, metric.ServerID, metric.ServerKey)

	return nil
}

func (s *PostgresStorage) updateServerLastSeen(ctx context.Context, serverID, serverKey string) {
	query := `
		INSERT INTO servers (server_id, server_key, last_seen)
		VALUES ($1, $2, NOW())
		ON CONFLICT (server_id) 
		DO UPDATE SET 
			last_seen = EXCLUDED.last_seen,
			server_key = EXCLUDED.server_key
	`

	s.db.ExecContext(ctx, query, serverID, serverKey)
}

func (s *PostgresStorage) GetLatestMetrics(ctx context.Context, serverID string) ([]*types.Metric, error) {
	query := `
		SELECT DISTINCT ON (metric_type)
			server_id, server_key, metric_type, value, tags, version, timestamp
		FROM metrics
		WHERE server_id = $1
			AND timestamp > NOW() - INTERVAL '1 hour'
		ORDER BY metric_type, timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*types.Metric
	for rows.Next() {
		metric := &types.Metric{}
		var tagsJSON []byte

		err := rows.Scan(
			&metric.ServerID,
			&metric.ServerKey,
			&metric.Type,
			&metric.Value,
			&tagsJSON,
			&metric.Version,
			&metric.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &metric.Tags); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal tags")
			}
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (s *PostgresStorage) GetMetricsHistory(ctx context.Context, serverID string, metricType string, from, to time.Time) ([]*types.Metric, error) {
	query := `
		SELECT server_id, server_key, metric_type, value, tags, version, timestamp
		FROM metrics
		WHERE server_id = $1
			AND ($2 = '' OR metric_type = $2)
			AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := s.db.QueryContext(ctx, query, serverID, metricType, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics history: %w", err)
	}
	defer rows.Close()

	var metrics []*types.Metric
	for rows.Next() {
		metric := &types.Metric{}
		var tagsJSON []byte

		err := rows.Scan(
			&metric.ServerID,
			&metric.ServerKey,
			&metric.Type,
			&metric.Value,
			&tagsJSON,
			&metric.Version,
			&metric.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &metric.Tags); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal tags")
			}
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (s *PostgresStorage) GetServers(ctx context.Context) ([]string, error) {
	query := `
		SELECT server_id
		FROM servers
		WHERE last_seen > NOW() - INTERVAL '24 hours'
		ORDER BY last_seen DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query servers: %w", err)
	}
	defer rows.Close()

	var servers []string
	for rows.Next() {
		var serverID string
		if err := rows.Scan(&serverID); err != nil {
			return nil, fmt.Errorf("failed to scan server: %w", err)
		}
		servers = append(servers, serverID)
	}

	return servers, nil
}

func (s *PostgresStorage) StoreDLQMessage(ctx context.Context, topic string, partition int, offset int64, message []byte, errorMsg string) error {
	query := `
		INSERT INTO dead_letter_queue (topic, partition, offset, message, error, attempts)
		VALUES ($1, $2, $3, $4, $5, 1)
	`

	_, err := s.db.ExecContext(ctx, query, topic, partition, offset, message, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to store DLQ message: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"topic":     topic,
		"partition": partition,
		"offset":    offset,
		"error":     errorMsg,
	}).Warn("Message stored in dead letter queue")

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
