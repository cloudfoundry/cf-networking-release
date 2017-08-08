package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	validator "gopkg.in/validator.v2"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

type InternalConfig struct {
	LogPrefix          string    `json:"log_prefix" validate:"nonzero"`
	ListenHost         string    `json:"listen_host" validate:"nonzero"`
	InternalListenPort int       `json:"internal_listen_port" validate:"nonzero"`
	DebugServerHost    string    `json:"debug_server_host" validate:"nonzero"`
	DebugServerPort    int       `json:"debug_server_port" validate:"nonzero"`
	CACertFile         string    `json:"ca_cert_file" validate:"nonzero"`
	ServerCertFile     string    `json:"server_cert_file" validate:"nonzero"`
	ServerKeyFile      string    `json:"server_key_file" validate:"nonzero"`
	Database           db.Config `json:"database" validate:"nonzero"`
	TagLength          int       `json:"tag_length" validate:"nonzero"`
	MetronAddress      string    `json:"metron_address" validate:"nonzero"`
	LogLevel           string    `json:"log_level"`
	RequestTimeout     int       `json:"request_timeout" validate:"min=1"`
}

func (c *InternalConfig) Validate() error {
	return validator.Validate(c)
}

func NewInternal(path string) (*InternalConfig, error) {
	jsonBytes, err := ioutil.ReadFile(path)
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
