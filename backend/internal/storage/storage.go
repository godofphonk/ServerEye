package storage

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/servereye/servereye/pkg/publisher"
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
