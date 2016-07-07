package config

type Config struct {
	PolicyServerURL   string `json:"policy_server_url"`
	PollInterval      int    `json:"poll_interval"`
	ListenHost        string `json:"listen_host"`
	ListenPort        int    `json:"listen_port"`
	VNI               int    `json:"vni"`
	FlannelSubnetFile string `json:"flannel_subnet_file"`
}
