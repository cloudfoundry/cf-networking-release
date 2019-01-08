package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"gopkg.in/validator.v2"
)

type Config struct {
	Address                           string   `json:"address" validate:"nonzero"`
	Port                              string   `json:"port" validate:"nonzero"`
	ServiceDiscoveryControllerAddress string   `json:"service_discovery_controller_address" validate:"nonzero"`
	ServiceDiscoveryControllerPort    string   `json:"service_discovery_controller_port" validate:"nonzero"`
	ClientCert                        string   `json:"client_cert" validate:"nonzero"`
	ClientKey                         string   `json:"client_key" validate:"nonzero"`
	CACert                            string   `json:"ca_cert" validate:"nonzero"`
	MetronPort                        int      `json:"metron_port" validate:"min=1"`
	MetricsEmitSeconds                int      `json:"metrics_emit_seconds" validate:"min=1"`
	LogLevelAddress                   string   `json:"log_level_address" validate:"nonzero"`
	LogLevelPort                      int      `json:"log_level_port" validate:"min=1"`
	InternalServiceMeshDomains        []string `json:"internal_service_mesh_domains"`
	InternalRouteVIPRange             string   `json:"internal_route_vip_range" validate:"cidr"`
}

func init() {
	validator.SetValidationFunc("cidr", func(v interface{}, param string) error {
		cidr, ok := v.(string)
		if !ok {
			return errors.New(fmt.Sprintf("Unable to cast expected cidr to string: %v", v))
		}

		_, _, err := net.ParseCIDR(cidr)
		return err
	})
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

func (c *Config) GetInternalRouteVIPRangeCIDR() *net.IPNet {
	// We can ignore the error because it's been validated
	_, cidr, _ := net.ParseCIDR(c.InternalRouteVIPRange)
	return cidr
}
