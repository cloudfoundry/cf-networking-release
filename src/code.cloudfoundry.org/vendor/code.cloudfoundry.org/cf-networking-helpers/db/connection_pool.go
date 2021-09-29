package db

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
)

func NewConnectionPool(conf Config,
	maxOpenConnections int, maxIdleConnections int, connMaxLifetime time.Duration,
	logPrefix string, jobPrefix string, logger lager.Logger,
) (*ConnWrapper, error) {

	retriableConnector := RetriableConnector{
		Logger:        logger,
		Connector:     GetConnectionPool,
		Sleeper:       SleeperFunc(time.Sleep),
		RetryInterval: time.Duration(3) * time.Second,
		MaxRetries:    10,
	}

	logger.Info("getting db connection", lager.Data{})
	timeoutCtx, _ := context.WithTimeout(context.Background(), time.Duration(conf.Timeout)*time.Second)
	connectionPool, err := retriableConnector.GetConnectionPool(conf, timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: db connect: %s", logPrefix, jobPrefix, err)
	}

	connectionPool.SetMaxOpenConns(maxOpenConnections)
	connectionPool.SetMaxIdleConns(maxIdleConnections)
	connectionPool.SetConnMaxLifetime(connMaxLifetime)
	logger.Info("db connection retrieved", lager.Data{})

	return connectionPool, nil
}
