package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/db"

	"code.cloudfoundry.org/lager"

	"gopkg.in/validator.v2"
)

type Config struct {
	ListenHost         string         `json:"listen_host" validate:"nonzero"`
	ListenPort         int            `json:"listen_port" validate:"nonzero"`
	InternalListenPort int            `json:"internal_listen_port" validate:"nonzero"`
	CACertFile         string         `json:"ca_cert_file" validate:"nonzero"`
	ServerCertFile     string         `json:"server_cert_file" validate:"nonzero"`
	ServerKeyFile      string         `json:"server_key_file" validate:"nonzero"`
	UAAClient          string         `json:"uaa_client" validate:"nonzero"`
	UAAClientSecret    string         `json:"uaa_client_secret" validate:"nonzero"`
	UAAURL             string         `json:"uaa_url" validate:"nonzero"`
	CCURL              string         `json:"cc_url" validate:"nonzero"`
	SkipSSLValidation  bool           `json:"skip_ssl_validation"`
	Database           db.Config      `json:"database" validate:"nonzero"`
	TagLength          int            `json:"tag_length" validate:"nonzero"`
	MetronAddress      string         `json:"metron_address" validate:"nonzero"`
	LogLevel           lager.LogLevel `json:"log_level"`
}

func (c *Config) Validate() error {
	return validator.Validate(c)
}

func New(path string) (*Config, error) {
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %s", err)
	}

	cfg := Config{
		LogLevel: lager.INFO,
	}
	err = json.Unmarshal(jsonBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %s", err)
	}

	if err := cfg.Validate(); err != nil {
		return &cfg, fmt.Errorf("invalid config: %s", err)
	}

	return &cfg, nil
}
