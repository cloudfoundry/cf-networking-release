package config

type Config struct {
	PolicyServerURL string `json:"policy_server_url"`
	PollInterval    int    `json:"poll_interval"`
	ListenHost      string `json:"listen_host"`
	ListenPort      int    `json:"listen_port"`
}
