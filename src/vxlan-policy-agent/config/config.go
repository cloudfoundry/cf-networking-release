package config

type VxlanPolicyAgent struct {
	PollInterval      int    `json:"poll_interval"`
	GardenAddress     string `json:"garden_address"`
	GardenProtocol    string `json:"garden_protocol"`
	PolicyServerURL   string `json:"policy_server_url"`
	VNI               int    `json:"vni"`
	FlannelSubnetFile string `json:"flannel_subnet_file"`
	MetronAddress     string `json:"metron_address"`
	ServerCACert      string `json:"ca_cert"`
	ClientCert        string `json:"server_cert"`
	ClientKey         string `json:"server_key"`
}
