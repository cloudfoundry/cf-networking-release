package db

import (
	"context"
	"time"

	"code.cloudfoundry.org/lager/v3"
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
	Logger        lager.Logger
	Connector     func(Config, context.Context) (*ConnWrapper, error)
	Sleeper       sleeper
	RetryInterval time.Duration
	MaxRetries    int
}

func (r *RetriableConnector) GetConnectionPool(dbConfig Config, ctx context.Context) (*ConnWrapper, error) {
	var attempts int
	for {
		attempts++

		db, err := r.Connector(dbConfig, ctx)
		if err == nil {
			return db, nil
		}

		if _, ok := err.(RetriableError); ok && attempts < r.MaxRetries {
			r.Logger.Info("retrying due to getting an error", lager.Data{
				"error": err,
			})
			r.Sleeper.Sleep(r.RetryInterval)
			continue
		}

		return nil, err
	}
}
