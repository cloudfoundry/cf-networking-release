package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/validator.v2"
)

type VxlanPolicyAgent struct {
	PollInterval         int    `json:"poll_interval" validate:"nonzero"`
	Datastore            string `json:"cni_datastore_path" validate:"nonzero"`
	PolicyServerURL      string `json:"policy_server_url" validate:"min=1"`
	VNI                  int    `json:"vni" validate:"nonzero"`
	MetronAddress        string `json:"metron_address" validate:"nonzero"`
	ServerCACertFile     string `json:"ca_cert_file" validate:"nonzero"`
	ClientCertFile       string `json:"client_cert_file" validate:"nonzero"`
	ClientKeyFile        string `json:"client_key_file" validate:"nonzero"`
	ClientTimeoutSeconds int    `json:"client_timeout_seconds" validate:"nonzero"`
	IPTablesLockFile     string `json:"iptables_lock_file" validate:"nonzero"`
	DebugServerHost      string `json:"debug_server_host" validate:"nonzero"`
	DebugServerPort      int    `json:"debug_server_port" validate:"nonzero"`
	LogLevel             string `json:"log_level"`
	IPTablesLogging      bool   `json:"iptables_c2c_logging"`
}

func (c *VxlanPolicyAgent) Validate() error {
	return validator.Validate(c)
}

func New(configFilePath string) (*VxlanPolicyAgent, error) {
	cfg := &VxlanPolicyAgent{}
	if _, err := os.Stat(configFilePath); err != nil {
		return cfg, fmt.Errorf("file does not exist: %s", err)
	}

	configBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return cfg, fmt.Errorf("reading config file: %s", err)
	}

	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parsing config (%s): %s", configFilePath, err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("invalid config: %s", err)
	}

	return cfg, nil
}
