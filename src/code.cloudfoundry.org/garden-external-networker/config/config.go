package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	CniPluginDir               string   `json:"cni_plugin_dir"`
	CniConfigDir               string   `json:"cni_config_dir"`
	BindMountDir               string   `json:"bind_mount_dir"`
	StateFilePath              string   `json:"state_file"`
	StartPort                  int      `json:"start_port"`
	TotalPorts                 int      `json:"total_ports"`
	LogPrefix                  string   `json:"log_prefix"`
	SearchDomains              []string `json:"search_domains"`
	IPTablesLockFile           string   `json:"iptables_lock_file"`
	ProxyRedirectCIDR          string   `json:"proxy_redirect_cidr"`
	ProxyPort                  int      `json:"proxy_port"`
	ProxyUID                   *int     `json:"proxy_uid"`
	EnableIngressProxyRedirect bool     `json:"enable_ingress_proxy_redirect"`
}

func New(configFilePath string) (Config, error) {
	cfg := Config{}

	if _, err := os.Stat(configFilePath); err != nil {
		return cfg, fmt.Errorf("file does not exist: %s", err)
	}

	configBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return cfg, fmt.Errorf("reading config file: %s", err)
	}

	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parsing config (%s): %s", configFilePath, err)
	}

	if cfg.CniPluginDir == "" {
		return cfg, fmt.Errorf("missing required config 'cni_plugin_dir'")
	}

	if cfg.CniConfigDir == "" {
		return cfg, fmt.Errorf("missing required config 'cni_config_dir'")
	}

	if cfg.BindMountDir == "" {
		return cfg, fmt.Errorf("missing required config 'bind_mount_dir'")
	}

	if cfg.StateFilePath == "" {
		return cfg, fmt.Errorf("missing required config 'state_file'")
	}

	if cfg.StartPort == 0 {
		return cfg, fmt.Errorf("missing required config 'start_port'")
	}

	if cfg.TotalPorts == 0 {
		return cfg, fmt.Errorf("missing required config 'total_ports'")
	}

	if cfg.LogPrefix == "" {
		return cfg, fmt.Errorf("missing required config 'log_prefix'")
	}

	if cfg.IPTablesLockFile == "" {
		return cfg, fmt.Errorf("missing required config 'iptables_lock_file'")
	}

	if cfg.ProxyPort == 0 {
		return cfg, fmt.Errorf("missing required config 'proxy_port'")
	}

	if cfg.ProxyUID == nil {
		return cfg, fmt.Errorf("missing required config 'proxy_uid'")
	}

	return cfg, nil
}
