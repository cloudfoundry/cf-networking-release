package config

type Natman struct {
	PollInterval   int    `json:"poll_interval"`
	GardenAddress  string `json:"garden_address"`
	GardenProtocol string `json:"garden_protocol"`
}
