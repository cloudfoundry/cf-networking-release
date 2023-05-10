package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	"code.cloudfoundry.org/cf-networking-helpers/db/monitor"
	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type RetriableError struct {
	Inner error
	Msg   string
}

func (r RetriableError) Error() string {
	return fmt.Sprintf("%s: %s", r.Msg, r.Inner.Error())
}

func GetConnectionPool(dbConfig Config, ctx context.Context) (*ConnWrapper, error) {
	connectionString, err := dbConfig.ConnectionString()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection string: %s", err)
	}
	nativeDBConn, err := sql.Open(dbConfig.Type, connectionString)
	if err != nil {
		return nil, fmt.Errorf("unable to open database connection: %s", err)
	}

	dbConn := sqlx.NewDb(nativeDBConn, dbConfig.Type)

	if err = dbConn.PingContext(ctx); err != nil {
		dbConn.Close()
		if netErr, ok := err.(*net.OpError); ok {
			return nil, RetriableError{
				Inner: netErr,
				Msg:   "unable to ping",
			}
		}
		return nil, fmt.Errorf("unable to ping: %s", err)
	}

	return &ConnWrapper{
		DB:      dbConn,
		Monitor: monitor.New(),
	}, nil
}
