package db

import (
	"database/sql"

	"code.cloudfoundry.org/cf-networking-helpers/db/monitor"
	"github.com/jmoiron/sqlx"
)

type ConnWrapper struct {
	*sqlx.DB
	Monitor monitor.Monitor
}

func (c *ConnWrapper) Beginx() (Transaction, error) {
	var innerTx *sqlx.Tx
	err := c.Monitor.Monitor(func() error {
		var err error
		innerTx, err = c.DB.Beginx()
		return err
	})

	tx := &monitoredTx{
		tx:      innerTx,
		monitor: c.Monitor,
	}

	return tx, err
}

func (c *ConnWrapper) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var result *sql.Rows
	err := c.Monitor.Monitor(func() error {
		var err error
		result, err = c.DB.Query(query, args...)
		return err
	})
	return result, err
}

func (c *ConnWrapper) QueryRow(query string, args ...interface{}) *sql.Row {
	var result *sql.Row
	c.Monitor.Monitor(func() error {
		result = c.DB.QueryRow(query, args...)
		return nil
	})
	return result
}

func (c *ConnWrapper) OpenConnections() int {
	return c.DB.Stats().OpenConnections
}

func (c *ConnWrapper) RawConnection() *sqlx.DB {
	return c.DB
}
