package config

type Config struct {
	FlannelSubnetFile string `json:"flannel_subnet_file"`
	BridgeName        string `json:"bridge_name"`
	MetronAddress     string `json:"metron_address"`
	MetadataFilename  string `json:"metadata_filename"`
}
