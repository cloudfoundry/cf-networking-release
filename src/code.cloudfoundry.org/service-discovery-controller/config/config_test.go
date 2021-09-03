package config_test

import (
	"crypto/x509"
	"encoding/json"
	"fmt"

	. "code.cloudfoundry.org/service-discovery-controller/config"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var configJSON []byte

	BeforeEach(func() {
		configJSON = []byte(`{
			"address":"example.com",
			"port":"80053",
			"index":"62",
			"log_level_address": "localhost",
			"log_level_port": 8012,
			"server_cert": "some_path_server_cert",
			"server_key": "some_path_server_key",
			"ca_cert": "some_path_ca_cert",
			"nats":{
				"hosts": [{
					"hostname": "a-nats-host",
					"port": 1
				},
				{
					"hostname": "b-nats-host",
					"port": 2
				}],
				"user": "a-nats-user",
				"pass": "a-nats-pass"
			},
			"staleness_threshold_seconds": 5,
			"pruning_interval_seconds": 3,
			"metrics_emit_seconds": 6,
			"metron_port": 8080,
			"resume_pruning_delay_seconds": 2,
			"warm_duration_seconds": 5
		}`)
	})

	Context("when created from valid JSON", func() {
		It("contains the values in the JSON", func() {
			parsedConfig, err := NewConfig(configJSON)
			Expect(err).ToNot(HaveOccurred())

			Expect(parsedConfig.Address).To(Equal("example.com"))
			Expect(parsedConfig.Port).To(Equal("80053"))
			Expect(parsedConfig.Index).To(Equal("62"))
			Expect(parsedConfig.LogLevelAddress).To(Equal("localhost"))
			Expect(parsedConfig.LogLevelPort).To(Equal(8012))
			Expect(parsedConfig.ServerCert).To(Equal("some_path_server_cert"))
			Expect(parsedConfig.ServerKey).To(Equal("some_path_server_key"))
			Expect(parsedConfig.CACert).To(Equal("some_path_ca_cert"))
			Expect(parsedConfig.Index).To(Equal("62"))
			Expect(parsedConfig.NatsServers()).To(ContainElement("nats://a-nats-user:a-nats-pass@a-nats-host:1"))
			Expect(parsedConfig.NatsServers()).To(ContainElement("nats://a-nats-user:a-nats-pass@b-nats-host:2"))
			Expect(parsedConfig.StalenessThresholdSeconds).To(Equal(5))
			Expect(parsedConfig.PruningIntervalSeconds).To(Equal(3))
			Expect(parsedConfig.MetricsEmitSeconds).To(Equal(6))
			Expect(parsedConfig.ResumePruningDelaySeconds).To(Equal(2))
			Expect(parsedConfig.WarmDurationSeconds).To(Equal(5))
		})
	})

	Context("when specifying to use NATS over TLS", func() {
		It("interprets the configuration correctly", func() {
			var configBeingAltered Config
			err := json.Unmarshal(configJSON, &configBeingAltered)
			Expect(err).ToNot(HaveOccurred())

			_, serverCertPath, _, serverCert := testhelpers.GenerateCaAndMutualTlsCerts()
			_, clientCertPath, clientKeyPath, clientCert := testhelpers.GenerateCaAndMutualTlsCerts()

			parsedCert, err := x509.ParseCertificate(serverCert.Certificate[0])
			Expect(err).ToNot(HaveOccurred())
			expectedSubject := parsedCert.RawSubject

			configBeingAltered.Nats = NatsConfig{
				Hosts: []NatsHost{
					{
						Hostname: "tls-nats-server-1",
						Port:     33,
					},
					{
						Hostname: "tls-nats-server-2",
						Port:     44,
					},
				},
				TLSEnabled: true,
				CACerts:    serverCertPath,
				CertChain:  clientCertPath,
				PrivateKey: clientKeyPath,
			}

			configJSON, err = json.Marshal(configBeingAltered)
			Expect(err).ToNot(HaveOccurred())

			parsedConfig, err := NewConfig(configJSON)
			Expect(err).ToNot(HaveOccurred())

			Expect(parsedConfig.NatsServers()).To(ConsistOf([]string{
				"nats://tls-nats-server-1:33",
				"nats://tls-nats-server-2:44",
			}))

			Expect(parsedConfig.Nats.TLSEnabled).To(Equal(true))

			Expect(parsedConfig.Nats.CAPool).ToNot(BeNil())
			poolSubjects := parsedConfig.Nats.CAPool.Subjects()
			Expect(string(poolSubjects[0])).To(Equal(string(expectedSubject)))

			Expect(parsedConfig.Nats.ClientAuthCertificate).To(Equal(clientCert))
		})
	})

	Context("when constructed with invalid JSON", func() {
		It("returns an error", func() {
			configJSON := []byte(`garbage`)
			_, err := NewConfig(configJSON)
			Expect(err).To(MatchError(ContainSubstring("unmarshal config")))
		})
	})

	var requiredFields map[string]interface{}
	BeforeEach(func() {
		requiredFields = map[string]interface{}{
			"address":                      "example.com",
			"port":                         "80053",
			"server_cert":                  "path_to_cert",
			"server_key":                   "path_to_key",
			"ca_cert":                      "path_to_ca_cert",
			"metron_port":                  8080,
			"staleness_threshold_seconds":  5,
			"pruning_interval_seconds":     3,
			"metrics_emit_seconds":         678,
			"resume_pruning_delay_seconds": 2,
			"warm_duration_seconds":        5,
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
		Entry("invalid staleness_threshold_seconds", "staleness_threshold_seconds", -2, "StalenessThresholdSeconds: less than min"),
		Entry("invalid pruning_interval_seconds", "pruning_interval_seconds", -2, "PruningIntervalSeconds: less than min"),
		Entry("invalid metrics_emit_seconds", "metrics_emit_seconds", -2, "MetricsEmitSeconds: less than min"),
		Entry("invalid address", "address", "", "Address: zero value"),
		Entry("invalid port", "port", "", "Port: zero value"),
		Entry("invalid server_cert", "server_cert", "", "ServerCert: zero value"),
		Entry("invalid server_key", "server_key", "", "ServerKey: zero value"),
		Entry("invalid ca_cert", "ca_cert", "", "CACert: zero value"),
		Entry("invalid resume_pruning_delay_seconds", "resume_pruning_delay_seconds", -1, "ResumePruningDelaySeconds: less than min"),
		Entry("invalid warm_duration_seconds", "warm_duration_seconds", -1, "WarmDurationSeconds: less than min"),
	)
})

func cloneMap(original map[string]interface{}) map[string]interface{} {
	new := map[string]interface{}{}
	for k, v := range original {
		new[k] = v
	}
	return new
}
