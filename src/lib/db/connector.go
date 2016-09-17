package db

import (
	"fmt"
	"net"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type RetriableError struct {
	Inner error
	Msg   string
}

func (r RetriableError) Error() string {
	return fmt.Sprintf("%s: %s", r.Msg, r.Inner.Error())
}

func GetConnectionPool(dbConfig Config) (*sqlx.DB, error) {
	var dbConn *sqlx.DB
	var err error

	dbConn, err = sqlx.Open(dbConfig.Type, dbConfig.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("unable to open database connection: %s", err)
	}

	if err = dbConn.Ping(); err != nil {
		dbConn.Close()
		if netErr, ok := err.(*net.OpError); ok {
			return nil, RetriableError{
				Inner: netErr,
				Msg:   "unable to ping",
			}
		}
		return nil, fmt.Errorf("unable to ping: %s", err)
	}

	return dbConn, nil
}
