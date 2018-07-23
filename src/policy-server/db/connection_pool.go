package db

import (
	"database/sql"
	"log"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/lager"
	"github.com/jmoiron/sqlx"
)

type ConnWrapper struct {
	sqlxDB *sqlx.DB
}

//go:generate counterfeiter -o fakes/transaction.go --fake-name Transaction . Transaction
type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
	Commit() error
	Rollback() error
	Rebind(string) string
	DriverName() string
}

func (c *ConnWrapper) Beginx() (Transaction, error) {
	return c.sqlxDB.Beginx()
}

func (c *ConnWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	return c.sqlxDB.Exec(query, args...)
}

func (c *ConnWrapper) NamedExec(query string, arg interface{}) (sql.Result, error) {
	return c.sqlxDB.NamedExec(query, arg)
}

func (c *ConnWrapper) Get(dest interface{}, query string, args ...interface{}) error {
	return c.sqlxDB.Get(dest, query, args...)
}

func (c *ConnWrapper) Select(dest interface{}, query string, args ...interface{}) error {
	return c.sqlxDB.Select(dest, query, args...)
}

func (c *ConnWrapper) QueryRow(query string, args ...interface{}) *sql.Row {
	return c.sqlxDB.QueryRow(query, args...)
}

func (c *ConnWrapper) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return c.sqlxDB.Query(query, args...)
}

func (c *ConnWrapper) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	return c.sqlxDB.Queryx(query, args...)
}

func (c *ConnWrapper) Rebind(query string) string {
	return c.sqlxDB.Rebind(query)
}

func (c *ConnWrapper) DriverName() string {
	return c.sqlxDB.DriverName()
}

func (c *ConnWrapper) RawConnection() *sqlx.DB {
	return c.sqlxDB
}

func (c *ConnWrapper) Close() error {
	return c.sqlxDB.Close()
}

func NewConnectionPool(conf db.Config, maxOpenConnections int, maxIdleConnections int, logPrefix string, jobPrefix string, logger lager.Logger) *ConnWrapper {
	retriableConnector := db.RetriableConnector{
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: time.Duration(3) * time.Second,
		MaxRetries:    10,
	}

	logger.Info("getting db connection", lager.Data{})
	type dbConnection struct {
		ConnectionPool *sqlx.DB
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
		log.Fatalf("%s.%s: db connection timeout", logPrefix, jobPrefix)
	}
	if connectionResult.Err != nil {
		log.Fatalf("%s.%s: db connect: %s", logPrefix, jobPrefix, connectionResult.Err) // not tested
	}

	connectionPool := connectionResult.ConnectionPool

	connectionPool.SetMaxOpenConns(maxOpenConnections)
	connectionPool.SetMaxIdleConns(maxIdleConnections)
	logger.Info("db connection retrived", lager.Data{})

	return &ConnWrapper{sqlxDB: connectionPool}
}
