package db

import (
	"crypto/tls"

	"github.com/go-sql-driver/mysql"
)

type MySQLAdapter struct{}

func (m MySQLAdapter) ParseDSN(dsn string) (cfg *mysql.Config, err error) {
	return mysql.ParseDSN(dsn)
}

func (m MySQLAdapter) RegisterTLSConfig(key string, config *tls.Config) error {
	return mysql.RegisterTLSConfig(key, config)
}
