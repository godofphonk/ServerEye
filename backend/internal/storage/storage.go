package storage

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	db     *sql.DB
	logger *logrus.Logger
}

func New(databaseURL string, logger *logrus.Logger) (*Storage, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Storage{
		db:     db,
		logger: logger,
	}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
