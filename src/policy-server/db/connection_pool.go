package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/lager"
)

func NewErroringConnectionPool(conf db.Config, maxOpenConnections int, maxIdleConnections int, connMaxLifetime time.Duration, logPrefix string, jobPrefix string, logger lager.Logger) (*db.ConnWrapper, error) {
	retriableConnector := db.RetriableConnector{
		Logger:        logger,
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: time.Duration(3) * time.Second,
		MaxRetries:    10,
	}

	logger.Info("getting db connection", lager.Data{})
	timeoutCtx, _ := context.WithTimeout(context.Background(), time.Duration(conf.Timeout)*time.Second)
	connectionPool, err := retriableConnector.GetConnectionPool(conf, timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("%s.%s: db connect: %s", logPrefix, jobPrefix, err) // not tested
	}

	connectionPool.SetMaxOpenConns(maxOpenConnections)
	connectionPool.SetMaxIdleConns(maxIdleConnections)
	connectionPool.SetConnMaxLifetime(connMaxLifetime)
	logger.Info("db connection retrived", lager.Data{})

	return connectionPool, nil
}

func NewConnectionPool(conf db.Config, maxOpenConnections int, maxIdleConnections int, connMaxLifetime time.Duration, logPrefix string, jobPrefix string, logger lager.Logger) *db.ConnWrapper {
	conn, err := NewErroringConnectionPool(conf, maxOpenConnections, maxIdleConnections, connMaxLifetime, logPrefix, jobPrefix, logger)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return conn
}
