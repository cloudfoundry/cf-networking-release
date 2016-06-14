package config

type Config struct {
	ListenHost        string `json:"listen_host"`
	ListenPort        int    `json:"listen_port"`
	UAAClient         string `json:"uaa_client"`
	UAAClientSecret   string `json:"uaa_client_secret"`
	UAAURL            string `json:"uaa_url"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	DatabaseURL       string `json:"database_url"`
}
