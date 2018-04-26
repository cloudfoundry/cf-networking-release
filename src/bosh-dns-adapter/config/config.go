package config

import (
	"encoding/json"
	"fmt"
	"gopkg.in/validator.v2"
)

type Config struct {
	Address                           string `json:"address" validate:"nonzero"`
	Port                              string `json:"port" validate:"nonzero"`
	ServiceDiscoveryControllerAddress string `json:"service_discovery_controller_address" validate:"nonzero"`
	ServiceDiscoveryControllerPort    string `json:"service_discovery_controller_port" validate:"nonzero"`
	ClientCert                        string `json:"client_cert" validate:"nonzero"`
	ClientKey                         string `json:"client_key" validate:"nonzero"`
	CACert                            string `json:"ca_cert" validate:"nonzero"`
	MetronPort                        int    `json:"metron_port" validate:"min=1"`
	MetricsEmitSeconds                int    `json:"metrics_emit_seconds" validate:"min=1"`
	LogLevelAddress                   string `json:"log_level_address" validate:"nonzero"`
	LogLevelPort                      int    `json:"log_level_port" validate:"min=1"`
}

func NewConfig(configJSON []byte) (*Config, error) {
	adapterConfig := &Config{}
	err := json.Unmarshal(configJSON, adapterConfig)

	if err != nil {
		return nil, err
	}

	if err = validator.Validate(adapterConfig); err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	return adapterConfig, err
}
