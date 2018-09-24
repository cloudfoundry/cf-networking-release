package db

import (
	"fmt"
	"log"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/lager"
)

func NewErroringConnectionPool(conf db.Config, maxOpenConnections int, maxIdleConnections int, connMaxLifetime time.Duration, logPrefix string, jobPrefix string, logger lager.Logger) (*db.ConnWrapper, error) {
	retriableConnector := db.RetriableConnector{
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: time.Duration(3) * time.Second,
		MaxRetries:    10,
	}

	logger.Info("getting db connection", lager.Data{})
	type dbConnection struct {
		ConnectionPool *db.ConnWrapper
		Err            error
	}

	channel := make(chan dbConnection)
	go func() {
		connection, err := retriableConnector.GetConnectionPool(conf)
		channel <- dbConnection{connection, err}
	}()
	var connectionResult dbConnection
	select {
	case connectionResult = <-channel:
	case <-time.After(time.Duration(conf.Timeout) * time.Second):
		return nil, fmt.Errorf("%s.%s: db connection timeout", logPrefix, jobPrefix)
	}
	if connectionResult.Err != nil {
		return nil, fmt.Errorf("%s.%s: db connect: %s", logPrefix, jobPrefix, connectionResult.Err) // not tested
	}

	connectionPool := connectionResult.ConnectionPool

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
