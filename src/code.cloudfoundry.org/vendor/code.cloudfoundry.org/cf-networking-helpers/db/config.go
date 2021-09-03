package db

import (
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	Type                   string `json:"type" validate:"nonzero"`
	User                   string `json:"user" validate:"nonzero"`
	Password               string `json:"password"`
	Host                   string `json:"host" validate:"nonzero"`
	Port                   uint16 `json:"port" validate:"nonzero"`
	Timeout                int    `json:"timeout" validate:"min=1"`
	DatabaseName           string `json:"database_name" validate:""`
	RequireSSL             bool   `json:"require_ssl" validate:""`
	CACert                 string `json:"ca_cert" validate:""`
	SkipHostnameValidation bool   `json:"skip_hostname_validation" validate:""`
}

func (c Config) ConnectionString() (string, error) {
	if c.Timeout < 1 {
		return "", fmt.Errorf("timeout must be at least 1 second: %d", c.Timeout)
	}
	switch c.Type {
	case "postgres":
		return buildPostgresConnectionString(c)
	case "mysql":
		mysqlConnectionStringBuilder := &MySQLConnectionStringBuilder{
			MySQLAdapter: &MySQLAdapter{},
		}
		return mysqlConnectionStringBuilder.Build(c)
	default:
		return "", fmt.Errorf("database type '%s' is not supported", c.Type)
	}
}

func buildPostgresConnectionString(c Config) (string, error) {
	ms := (time.Duration(c.Timeout) * time.Second).Nanoseconds() / 1000 / 1000
	sslmode := "disable"
	params := url.Values{}

	if c.RequireSSL {
		if c.SkipHostnameValidation {
			sslmode = "require"
		} else {
			if c.CACert == "" {
				return "", fmt.Errorf("SSL is required but `CACert` is not provided")
			}
			sslmode = "verify-full"
			params.Add("sslrootcert", c.CACert)
		}
	}

	params.Add("sslmode", sslmode)
	params.Add("connect_timeout", fmt.Sprintf("%d", ms))

	connURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:     c.DatabaseName,
		RawQuery: params.Encode(),
	}
	return connURL.String(), nil
}
