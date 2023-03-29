package config

import (
	"encoding/json"
	"os"

	"code.cloudfoundry.org/debugserver"
	loggingclient "code.cloudfoundry.org/diego-logging-client"
	"code.cloudfoundry.org/durationjson"
	"code.cloudfoundry.org/lager/v3/lagerflags"
)

type LocketConfig struct {
	CaFile                        string                `json:"ca_file"`
	CertFile                      string                `json:"cert_file"`
	DatabaseConnectionString      string                `json:"database_connection_string"`
	MaxOpenDatabaseConnections    int                   `json:"max_open_database_connections,omitempty"`
	MaxDatabaseConnectionLifetime durationjson.Duration `json:"max_database_connection_lifetime,omitempty"`
	DatabaseDriver                string                `json:"database_driver,omitempty"`
	KeyFile                       string                `json:"key_file"`
	ListenAddress                 string                `json:"listen_address"`
	SQLCACertFile                 string                `json:"sql_ca_cert_file,omitempty"`
	SQLEnableIdentityVerification bool                  `json:"sql_enable_identity_verification,omitempty"`
	LoggregatorConfig             loggingclient.Config  `json:"loggregator"`
	ReportInterval                durationjson.Duration `json:"report_interval,omitempty"`
	debugserver.DebugServerConfig
	lagerflags.LagerConfig
}

func NewLocketConfig(configPath string) (LocketConfig, error) {
	locketConfig := LocketConfig{}
	configFile, err := os.Open(configPath)
	if err != nil {
		return LocketConfig{}, err
	}

	defer configFile.Close()

	decoder := json.NewDecoder(configFile)

	err = decoder.Decode(&locketConfig)
	if err != nil {
		return LocketConfig{}, err
	}

	return locketConfig, nil
}
