package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/validator.v2"
)

type Config struct {
	KernelLogFile         string `json:"kernel_log_file" validate:"nonzero"`
	ContainerMetadataFile string `json:"container_metadata_file" validate:"nonzero"`
	OutputLogFile         string `json:"output_log_file" validate:"nonzero"`
	MetronAddress         string `json:"metron_address" validate:"nonzero"`
	HostIp                string `json:"host_ip" validate:"nonzero"`
	HostGuid              string `json:"host_guid" validate:"nonzero"`
}

func New(path string) (*Config, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("file does not exist: %s", err)
	}
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %s", err)
	}

	cfg := Config{}
	err = json.Unmarshal(jsonBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %s", err)
	}

	if err := validator.Validate(cfg); err != nil {
		return &cfg, fmt.Errorf("invalid config: %s", err)
	}

	return &cfg, nil
}
