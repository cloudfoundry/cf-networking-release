package config

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/lager"
)

type Netmon struct {
	PollInterval  int    `json:"poll_interval"`
	MetronAddress string `json:"metron_address"`
	InterfaceName string `json:"interface_name"`
	LogLevel      string `json:"log_level"`
}

func (n Netmon) ParseLogLevel() (lager.LogLevel, error) {
	switch strings.ToLower(n.LogLevel) {
	case "debug":
		return 0, nil
	case "info":
		return 1, nil
	case "error":
		return 2, nil
	case "fatal":
		return 3, nil
	}
	return 0, fmt.Errorf(`unknown log level %q`, n.LogLevel)
}
