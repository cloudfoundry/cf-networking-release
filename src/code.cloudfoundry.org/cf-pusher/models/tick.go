package models

type Manifest struct {
	Applications []Application `yaml:"applications,omitempty"`
}

type Application struct {
	Name      string      `yaml:"name,omitempty"`
	Command   string      `yaml:"command,omitempty"`
	Memory    string      `yaml:"memory,omitempty"`
	DiskQuota string      `yaml:"disk_quota,omitempty"`
	BuildPack string      `yaml:"buildpack,omitempty"`
	Instances int         `yaml:"instances,omitempty"`
	Env       interface{} `yaml:"env,omitempty"`
}

type TickEnvironment struct {
	GoPackageName      string `yaml:"GOPACKAGENAME,omitempty"`
	RegistryBaseURL    string `yaml:"REGISTRY_BASE_URL,omitempty"`
	RegistryTTLSeconds int    `yaml:"REGISTRY_TTL_SECONDS,omitempty"`
	StartPort          int    `yaml:"START_PORT,omitempty"`
	ListenPorts        int    `yaml:"LISTEN_PORTS,omitempty"`
}

type ProxyEnvironment struct {
	GoPackageName string `yaml:"GOPACKAGENAME,omitempty"`
}
