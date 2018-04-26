package config

import (
	"encoding/json"
	"fmt"
	"net/url"

	"gopkg.in/validator.v2"
)

type Config struct {
	Address                   string       `json:"address" validate:"nonzero"`
	Port                      string       `json:"port" validate:"nonzero"`
	Nats                      []NatsConfig `json:"nats"`
	Index                     string       `json:"index"`
	ServerCert                string       `json:"server_cert" validate:"nonzero"`
	ServerKey                 string       `json:"server_key" validate:"nonzero"`
	CACert                    string       `json:"ca_cert" validate:"nonzero"`
	MetronPort                int          `json:"metron_port" validate:"min=1"`
	LogLevelAddress           string       `json:"log_level_address"`
	LogLevelPort              int          `json:"log_level_port"`
	StalenessThresholdSeconds int          `json:"staleness_threshold_seconds" validate:"min=1"`
	PruningIntervalSeconds    int          `json:"pruning_interval_seconds" validate:"min=1"`
	MetricsEmitSeconds        int          `json:"metrics_emit_seconds" validate:"min=1"`
	ResumePruningDelaySeconds int          `json:"resume_pruning_delay_seconds" validate:"min=0"`
	WarmDurationSeconds       int          `json:"warm_duration_seconds" validate:"min=0"`
}

type NatsConfig struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

func NewConfig(configJSON []byte) (*Config, error) {
	sdcConfig := &Config{}
	err := json.Unmarshal(configJSON, sdcConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %s", err)
	}

	if err = validator.Validate(sdcConfig); err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}
	return sdcConfig, err
}

func (c *Config) NatsServers() []string {
	var natsServers []string
	for _, info := range c.Nats {
		uri :=
			url.URL{
				Scheme: "nats",
				User:   url.UserPassword(info.User, info.Pass),
				Host:   fmt.Sprintf("%s:%d", info.Host, info.Port),
			}
		natsServers = append(natsServers, uri.String())

	}

	return natsServers
}
