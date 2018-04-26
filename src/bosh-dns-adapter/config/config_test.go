package config_test

import (
	. "bosh-dns-adapter/config"

	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("when created from valid JSON", func() {
		It("contains the values in the JSON", func() {
			configJSON := []byte(`{
				"address":"example.com",
				"port":"80053",
				"service_discovery_controller_address":"bar.com",
				"service_discovery_controller_port":"80055",
				"client_cert": "client.cert",
				"client_key": "client.key",
				"ca_cert": "ca.cert",
				"metrics_emit_seconds": 6,
				"metron_port": 8080,
				"log_level_address": "log-level-address",
				"log_level_port": 9090

			}`)

			parsedConfig, err := NewConfig(configJSON)
			Expect(err).ToNot(HaveOccurred())

			Expect(parsedConfig.Address).To(Equal("example.com"))
			Expect(parsedConfig.Port).To(Equal("80053"))
			Expect(parsedConfig.ServiceDiscoveryControllerAddress).To(Equal("bar.com"))
			Expect(parsedConfig.ServiceDiscoveryControllerPort).To(Equal("80055"))
			Expect(parsedConfig.ClientCert).To(Equal("client.cert"))
			Expect(parsedConfig.ClientKey).To(Equal("client.key"))
			Expect(parsedConfig.CACert).To(Equal("ca.cert"))

			Expect(parsedConfig.MetricsEmitSeconds).To(Equal(6))
			Expect(parsedConfig.MetronPort).To(Equal(8080))
			Expect(parsedConfig.LogLevelAddress).To(Equal("log-level-address"))
			Expect(parsedConfig.LogLevelPort).To(Equal(9090))
		})
	})

	Context("when constructed with invalid JSON", func() {
		It("returns an error", func() {
			configJSON := []byte(`garbage`)
			_, err := NewConfig(configJSON)
			Expect(err).To(HaveOccurred())
		})
	})

	var requiredFields map[string]interface{}
	BeforeEach(func() {
		requiredFields = map[string]interface{}{
			"address": "example.com",
			"port":    "80053",
			"service_discovery_controller_address": "example.com",
			"service_discovery_controller_port":    "80053",
			"client_cert":                          "path_to_cert",
			"client_key":                           "path_to_key",
			"ca_cert":                              "path_to_ca_cert",
			"metron_port":                          8080,
			"metrics_emit_seconds":                 678,
			"log_level_address":                    "log_level_address",
			"log_level_port":                       8081,
		}
	})

	DescribeTable("when config file field contains an invalid value",
		func(invalidField string, value interface{}, errorString string) {
			cfg := cloneMap(requiredFields)
			cfg[invalidField] = value

			cfgBytes, _ := json.Marshal(cfg)
			_, err := NewConfig(cfgBytes)

			Expect(err).To(MatchError(fmt.Sprintf("invalid config: %s", errorString)))
		},

		Entry("invalid metron_port", "metron_port", -2, "MetronPort: less than min"),
		Entry("invalid metrics_emit_seconds", "metrics_emit_seconds", -2, "MetricsEmitSeconds: less than min"),
		Entry("invalid address", "address", "", "Address: zero value"),
		Entry("invalid service_discovery_controller_address", "service_discovery_controller_address", "", "ServiceDiscoveryControllerAddress: zero value"),
		Entry("invalid port", "port", "", "Port: zero value"),
		Entry("invalid service_discovery_controller_port", "service_discovery_controller_port", "", "ServiceDiscoveryControllerPort: zero value"),
		Entry("invalid client_cert", "client_cert", "", "ClientCert: zero value"),
		Entry("invalid client_key", "client_key", "", "ClientKey: zero value"),
		Entry("invalid ca_cert", "ca_cert", "", "CACert: zero value"),
		Entry("invalid log_level_address", "log_level_address", "", "LogLevelAddress: zero value"),
		Entry("invalid log_level_port", "log_level_port", -2, "LogLevelPort: less than min"),
	)
})

func cloneMap(original map[string]interface{}) map[string]interface{} {
	new := map[string]interface{}{}
	for k, v := range original {
		new[k] = v
	}
	return new
}
