package config

import (
	"encoding/json"
	"fmt"
	"os"

	validator "gopkg.in/validator.v2"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/locket"
)

type ASGSyncerConfig struct {
	ASGSyncInterval      int       `json:"asg_poll_interval_seconds" validate:"min=0"`
	RetryDeadline        int       `json:"retry_deadline_seconds" validate:"min=1"`
	UUID                 string    `json:"uuid" validate:"nonzero"`
	Database             db.Config `json:"database" validate:"nonzero"`
	UAAClient            string    `json:"uaa_client" validate:"nonzero"`
	UAAClientSecret      string    `json:"uaa_client_secret" validate:"nonzero"`
	UAACA                string    `json:"uaa_ca"`
	UAAURL               string    `json:"uaa_url" validate:"nonzero"`
	UAAPort              int       `json:"uaa_port" validate:"nonzero"`
	CCURL                string    `json:"cc_url" validate:"nonzero"`
	CCInternalURL        string    `json:"cc_internal_url" validate:"nonzero"`
	CCCA                 string    `json:"cc_ca_cert"`
	CCInternalCA         string    `json:"cc_internal_ca_cert"`
	CCInternalClientCert string    `json:"cc_internal_client_cert"`
	CCInternalClientKey  string    `json:"cc_internal_client_key"`
	LogLevel             string    `json:"log_level"`
	LogPrefix            string    `json:"log_prefix" validate:"nonzero"`
	MetronAddress        string    `json:"metron_address" validate:"nonzero"`
	SkipSSLValidation    bool      `json:"skip_ssl_validation"`
	locket.ClientLocketConfig
}

func (c *ASGSyncerConfig) Validate() error {
	return validator.Validate(c)
}

func NewASGSyncer(path string) (*ASGSyncerConfig, error) {
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %s", err)
	}

	cfg := ASGSyncerConfig{}
	err = json.Unmarshal(jsonBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %s", err)
	}

	if err := cfg.Validate(); err != nil {
		return &cfg, fmt.Errorf("invalid config: %s", err)
	}

	return &cfg, nil
}
