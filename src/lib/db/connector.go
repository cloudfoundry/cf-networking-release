package db

import (
	"fmt"
	"net"

	"github.com/jmoiron/sqlx"
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

	configString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Name, dbConfig.SSLMode)
	dbConn, err = sqlx.Open("postgres", configString)
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
