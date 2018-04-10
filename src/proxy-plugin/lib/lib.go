package lib

import (
	"encoding/json"
	"fmt"
)

type ProxyConfig struct {
	OverlayNetwork string `json:"overlay_network"`
	ProxyPort      int    `json:"proxy_port"`
}

func LoadProxyConfig(bytes []byte) (*ProxyConfig, error) {
	n := &ProxyConfig{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("loading proxy config: %v", err)
	}

	return n, nil
}
