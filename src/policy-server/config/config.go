package config

import "lib/db"

type Config struct {
	ListenHost        string    `json:"listen_host"`
	ListenPort        int       `json:"listen_port"`
	UAAClient         string    `json:"uaa_client"`
	UAAClientSecret   string    `json:"uaa_client_secret"`
	UAAURL            string    `json:"uaa_url"`
	SkipSSLValidation bool      `json:"skip_ssl_validation"`
	Database          db.Config `json:"database"`
	TagLength         int       `json:"tag_length"`
}
