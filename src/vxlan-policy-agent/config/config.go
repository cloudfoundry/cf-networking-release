package config

type VxlanPolicyAgent struct {
	PollInterval      int    `json:"poll_interval"`
	Datastore         string `json:"datastore"`
	PolicyServerURL   string `json:"policy_server_url"`
	VNI               int    `json:"vni"`
	FlannelSubnetFile string `json:"flannel_subnet_file"`
	MetronAddress     string `json:"metron_address"`
	ServerCACertFile  string `json:"ca_cert_file"`
	ClientCertFile    string `json:"client_cert_file"`
	ClientKeyFile     string `json:"client_key_file"`
}
