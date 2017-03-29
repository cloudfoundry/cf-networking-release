package config

type Config struct {
	FlannelSubnetFile string `json:"flannel_subnet_file"`
	MetronAddress     string `json:"metron_address"`
	MetadataFilename  string `json:"metadata_filename"`
	HealthCheckPort   int    `json:"health_check_port"`
}
