package db

import (
	"database/sql"

	"code.cloudfoundry.org/cf-networking-helpers/db/monitor"
	"github.com/jmoiron/sqlx"
)

//go:generate counterfeiter -o fakes/transaction.go --fake-name Transaction . Transaction
type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) RowScanner
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
	Commit() error
	Rollback() error
	Rebind(string) string
	DriverName() string
}

type RowScanner interface {
	Scan(dest ...interface{}) error
}

type monitoredTx struct {
	tx      *sqlx.Tx
	monitor monitor.Monitor
}

func (tx *monitoredTx) Exec(query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	err := tx.monitor.Monitor(func() error {
		var err error
		result, err = tx.tx.Exec(query, args...)
		return err
	})
	return result, err
}

func (tx *monitoredTx) QueryRow(query string, args ...interface{}) RowScanner {
	return NewRowScanner(tx.monitor, tx.tx.QueryRow(query, args...))
}

func (tx *monitoredTx) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	var result *sqlx.Rows
	err := tx.monitor.Monitor(func() error {
		var err error
		result, err = tx.tx.Queryx(query, args...)
		return err
	})
	return result, err
}

func (tx *monitoredTx) Commit() error {
	return tx.monitor.Monitor(tx.tx.Commit)
}

func (tx *monitoredTx) Rollback() error {
	return tx.monitor.Monitor(tx.tx.Rollback)
}

func (tx *monitoredTx) Rebind(query string) string {
	var result string
	tx.monitor.Monitor(func() error {
		result = tx.tx.Rebind(query)
		return nil
	})
	return result
}

func (tx *monitoredTx) DriverName() string {
	var result string
	tx.monitor.Monitor(func() error {
		result = tx.tx.DriverName()
		return nil
	})
	return result
}

type scannableRow struct {
	monitor monitor.Monitor
	scanner RowScanner
}

func NewRowScanner(monitor monitor.Monitor, scanner RowScanner) RowScanner {
	return &scannableRow{monitor: monitor, scanner: scanner}
}

func (r *scannableRow) Scan(dest ...interface{}) error {
	return r.monitor.Monitor(func() error {
		return r.scanner.Scan(dest...)
	})
}
