package cli_plugin

import (
	"time"
	"os"
	"strconv"
)

const defaultTimeout = 3 * time.Second

type dialTimeoutProvider struct{}

func (dialTimeoutProvider) Get() time.Duration {
	if envValue, found := os.LookupEnv("CF_DIAL_TIMEOUT"); found {
		envTimeoutValue, err := strconv.Atoi(envValue)
		if err != nil {
			return defaultTimeout
		}
		return time.Duration(envTimeoutValue) * time.Second
	}
	return defaultTimeout
}
