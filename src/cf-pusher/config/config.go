package config

type Config struct {
	ASGSize                            int      `json:"asg_size"`
	AdminPassword                      string   `json:"admin_password"`
	AdminSecret                        string   `json:"admin_secret"`
	AdminUser                          string   `json:"admin_user"`
	Api                                string   `json:"api"`
	AppInstances                       int      `json:"test_app_instances"`
	AppRegistryTTLSeconds              int      `json:"test_app_registry_ttl_seconds"`
	Applications                       int      `json:"test_applications"`
	AppsDomain                         string   `json:"apps_domain"`
	Concurrency                        int      `json:"concurrency"`
	DefaultSecurityGroups              []string `json:"default_security_groups"`
	ExtraListenPorts                   int      `json:"extra_listen_ports"`
	Internetless                       bool     `json:"internetless"`
	PolicyUpdateWaitSeconds            int      `json:"policy_update_wait_seconds"`
	Prefix                             string   `json:"prefix"`
	ProxyApplications                  int      `json:"proxy_applications"`
	ProxyInstances                     int      `json:"proxy_instances"`
	SamplePercent                      int      `json:"sample_percent"`
	SkipICMPTests                      bool     `json:"skip_icmp_tests"`
	RunCustomIPTablesCompatibilityTest bool     `json:"run_custom_iptables_compatibility_test"`
	SkipSearchDomainTests              bool     `json:"skip_search_domain_tests"`
	SkipSSLValidation                  bool     `json:"skip_ssl_validation"`
	SkipExperimentalDynamicEgressTest  bool     `json:"skip_experimental_dynamic_egress_tests"`
}
