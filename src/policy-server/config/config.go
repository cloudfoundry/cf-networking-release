package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/db"
)

type Config struct {
	ListenHost         string    `json:"listen_host"`
	ListenPort         int       `json:"listen_port"`
	InternalListenPort int       `json:"internal_listen_port"`
	CACertFile         string    `json:"ca_cert_file"`
	ServerCertFile     string    `json:"server_cert_file"`
	ServerKeyFile      string    `json:"server_key_file"`
	UAAClient          string    `json:"uaa_client"`
	UAAClientSecret    string    `json:"uaa_client_secret"`
	UAAURL             string    `json:"uaa_url"`
	CCURL              string    `json:"cc_url"`
	SkipSSLValidation  bool      `json:"skip_ssl_validation"`
	Database           db.Config `json:"database"`
	TagLength          int       `json:"tag_length"`
	MetronAddress      string    `json:"metron_address"`
}

func New(path string) (*Config, error) {
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %s", err)
	}

	var cfg Config
	err = json.Unmarshal(jsonBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %s", err)
	}

	return &cfg, nil
}
