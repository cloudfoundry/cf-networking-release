package db

import (
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

//go:generate counterfeiter -o ../fakes/sleeper.go --fake-name Sleeper . sleeper
type sleeper interface {
	Sleep(time.Duration)
}

type SleeperFunc func(time.Duration)

func (sf SleeperFunc) Sleep(duration time.Duration) {
	sf(duration)
}

type RetriableConnector struct {
	Connector     func(databaseConfig string) (*sqlx.DB, error)
	Sleeper       sleeper
	RetryInterval time.Duration
	MaxRetries    int
}

func (r *RetriableConnector) GetConnectionPool(databaseConfig string) (*sqlx.DB, error) {
	var attempts int
	for {
		attempts++

		db, err := r.Connector(databaseConfig)
		if err == nil {
			return db, nil
		}

		if _, ok := err.(RetriableError); ok && attempts < r.MaxRetries {
			r.Sleeper.Sleep(r.RetryInterval)
			continue
		}

		return nil, err
	}
}
