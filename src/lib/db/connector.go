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

	var connectionString string
	if dbConfig.Type == "mysql" {
		connectionString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Name)
	} else if dbConfig.Type == "postgres" {
		connectionString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Name, dbConfig.SSLMode)
	} else {
		panic("unknown db type " + dbConfig.Type)
	}

	dbConn, err = sqlx.Open(dbConfig.Type, connectionString)
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
