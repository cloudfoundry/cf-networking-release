package config

import (
	"encoding/json"
	"fmt"
	"os"

	validator "gopkg.in/validator.v2"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

type InternalConfig struct {
	LogPrefix                     string    `json:"log_prefix" validate:"nonzero"`
	ListenHost                    string    `json:"listen_host" validate:"nonzero"`
	InternalListenPort            int       `json:"internal_listen_port" validate:"nonzero"`
	DebugServerHost               string    `json:"debug_server_host" validate:"nonzero"`
	DebugServerPort               int       `json:"debug_server_port" validate:"nonzero"`
	HealthCheckPort               int       `json:"health_check_port" validate:"nonzero"`
	CACertFile                    string    `json:"ca_cert_file" validate:"nonzero"`
	ServerCertFile                string    `json:"server_cert_file" validate:"nonzero"`
	ServerKeyFile                 string    `json:"server_key_file" validate:"nonzero"`
	Database                      db.Config `json:"database" validate:"nonzero"`
	TagLength                     int       `json:"tag_length" validate:"nonzero"`
	MetronAddress                 string    `json:"metron_address" validate:"nonzero"`
	LogLevel                      string    `json:"log_level"`
	MaxIdleConnections            int       `json:"max_idle_connections" validate:"min=0"`
	MaxOpenConnections            int       `json:"max_open_connections" validate:"min=0"`
	MaxConnectionsLifetimeSeconds int       `json:"connections_max_lifetime_seconds" validate:"min=0"`
	UAAClient                     string    `json:"uaa_client" validate:"nonzero"`
	UAAClientSecret               string    `json:"uaa_client_secret" validate:"nonzero"`
	UAACA                         string    `json:"uaa_ca"`
	UAAURL                        string    `json:"uaa_url" validate:"nonzero"`
	UAAPort                       int       `json:"uaa_port" validate:"nonzero"`
	CCURL                         string    `json:"cc_url" validate:"nonzero"`
	CCCA                          string    `json:"cc_ca_cert"`
	SkipSSLValidation             bool      `json:"skip_ssl_validation"`
}

func (c *InternalConfig) Validate() error {
	return validator.Validate(c)
}

func NewInternal(path string) (*InternalConfig, error) {
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %s", err)
	}

	cfg := InternalConfig{}
	err = json.Unmarshal(jsonBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %s", err)
	}

	if err := cfg.Validate(); err != nil {
		return &cfg, fmt.Errorf("invalid config: %s", err)
	}

	return &cfg, nil
}
