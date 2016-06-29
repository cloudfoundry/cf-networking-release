package config

type Config struct {
	PolicyServerURL string `json:"policy_server_url"`
	PollInterval    int    `json:"poll_interval"`
}
