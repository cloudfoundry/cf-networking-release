package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"gopkg.in/validator.v2"
)

type Config struct {
	Address                   string     `json:"address" validate:"nonzero"`
	Port                      string     `json:"port" validate:"nonzero"`
	Nats                      NatsConfig `json:"nats"`
	Index                     string     `json:"index"`
	ServerCert                string     `json:"server_cert" validate:"nonzero"`
	ServerKey                 string     `json:"server_key" validate:"nonzero"`
	CACert                    string     `json:"ca_cert" validate:"nonzero"`
	MetronPort                int        `json:"metron_port" validate:"min=1"`
	LogLevelAddress           string     `json:"log_level_address"`
	LogLevelPort              int        `json:"log_level_port"`
	StalenessThresholdSeconds int        `json:"staleness_threshold_seconds" validate:"min=1"`
	PruningIntervalSeconds    int        `json:"pruning_interval_seconds" validate:"min=1"`
	MetricsEmitSeconds        int        `json:"metrics_emit_seconds" validate:"min=1"`
	ResumePruningDelaySeconds int        `json:"resume_pruning_delay_seconds" validate:"min=0"`
	WarmDurationSeconds       int        `json:"warm_duration_seconds" validate:"min=0"`
}

type NatsConfig struct {
	Hosts                 []NatsHost      `json:"hosts"`
	User                  string          `json:"user"`
	Pass                  string          `json:"pass"`
	TLSEnabled            bool            `json:"tls_enabled"`
	CACerts               string          `json:"ca_certs"`
	CAPool                *x509.CertPool  `json:"-"`
	CertChain             string          `json:"cert_chain"`
	PrivateKey            string          `json:"private_key"`
	ClientAuthCertificate tls.Certificate `json:"-"`
}

type NatsHost struct {
	Hostname string `json:"hostname"`
	Port     uint16 `json:"port"`
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

	if sdcConfig.Nats.TLSEnabled {
		certPool := x509.NewCertPool()
		caCerts, err := ioutil.ReadFile(sdcConfig.Nats.CACerts)
		if err != nil {
			return nil, fmt.Errorf("error reading NATS CA certs: %w", err)
		}
		if ok := certPool.AppendCertsFromPEM(caCerts); !ok {
			return nil, fmt.Errorf("unable to build NATS CA pool")
		}
		sdcConfig.Nats.CAPool = certPool

		certificate, err := tls.LoadX509KeyPair(sdcConfig.Nats.CertChain, sdcConfig.Nats.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("error reading NATS mTLS client auth: %w", err)
		}
		sdcConfig.Nats.ClientAuthCertificate = certificate
	}

	return sdcConfig, err
}

func (c *Config) NatsServers() []string {
	var natsServers []string
	for _, host := range c.Nats.Hosts {
		uri := url.URL{
			Scheme: "nats",
			Host:   fmt.Sprintf("%s:%d", host.Hostname, host.Port),
		}
		if c.Nats.User != "" || c.Nats.Pass != "" {
			uri.User = url.UserPassword(c.Nats.User, c.Nats.Pass)
		}
		natsServers = append(natsServers, uri.String())
	}
	return natsServers
}
