package config

import "lib/db"

type Config struct {
	ListenHost         string    `json:"listen_host"`
	ListenPort         int       `json:"listen_port"`
	InternalListenPort int       `json:"internal_listen_port"`
	CACert             []byte    `json:"ca_cert"`
	ServerCert         []byte    `json:"server_cert"`
	ServerKey          []byte    `json:"server_key"`
	UAAClient          string    `json:"uaa_client"`
	UAAClientSecret    string    `json:"uaa_client_secret"`
	UAAURL             string    `json:"uaa_url"`
	SkipSSLValidation  bool      `json:"skip_ssl_validation"`
	Database           db.Config `json:"database"`
	TagLength          int       `json:"tag_length"`
	MetronAddress      string    `json:"metron_address"`
}
