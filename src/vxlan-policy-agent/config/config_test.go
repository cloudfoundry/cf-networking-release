package config_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"vxlan-policy-agent/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("New", func() {
		var (
			file *os.File
			err  error
		)

		BeforeEach(func() {
			file, err = ioutil.TempFile(os.TempDir(), "config-")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when config file is valid", func() {
			It("returns the config", func() {
				file.WriteString(`{
					"poll_interval": 1234,
					"cni_datastore_path": "/some/datastore/path",
					"policy_server_url": "https://some-url:1234",
					"vni": 42,
					"metron_address": "http://1.2.3.4:1234",
					"ca_cert_file": "/some/ca/file",
					"client_cert_file": "/some/client/cert/file",
					"client_key_file": "/some/client/key/file",
					"iptables_lock_file":  "/var/vcap/data/lock",
					"debug_server_host": "http://5.6.7.8",
					"debug_server_port": 5678,
					"log_level": "debug",
					"log_prefix": "cfnetworking",
					"iptables_c2c_logging": true,
					"client_timeout_seconds":5,
					"iptables_accepted_udp_logs_per_sec":4
				}`)
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.PollInterval).To(Equal(1234))
				Expect(c.Datastore).To(Equal("/some/datastore/path"))
				Expect(c.PolicyServerURL).To(Equal("https://some-url:1234"))
				Expect(c.VNI).To(Equal(42))
				Expect(c.MetronAddress).To(Equal("http://1.2.3.4:1234"))
				Expect(c.ServerCACertFile).To(Equal("/some/ca/file"))
				Expect(c.ClientCertFile).To(Equal("/some/client/cert/file"))
				Expect(c.ClientKeyFile).To(Equal("/some/client/key/file"))
				Expect(c.IPTablesLockFile).To(Equal("/var/vcap/data/lock"))
				Expect(c.DebugServerHost).To(Equal("http://5.6.7.8"))
				Expect(c.DebugServerPort).To(Equal(5678))
				Expect(c.LogLevel).To(Equal("debug"))
				Expect(c.LogPrefix).To(Equal("cfnetworking"))
				Expect(c.IPTablesLogging).To(Equal(true))
				Expect(c.ClientTimeoutSeconds).To(Equal(5))
				Expect(c.IPTablesAcceptedUDPLogsPerSec).To(Equal(4))
			})
		})

		Context("when config file path does not exist", func() {
			It("returns the error", func() {
				_, err := config.New("not-exists")
				Expect(err).To(MatchError(ContainSubstring("file does not exist:")))
			})
		})

		Context("when config file is bad format", func() {
			It("returns the error", func() {
				file.WriteString("bad-format")
				_, err = config.New(file.Name())
				Expect(err).To(MatchError(ContainSubstring("parsing config")))
			})
		})

		Context("when config file contents blank", func() {
			It("returns the error", func() {
				_, err = config.New(file.Name())
				Expect(err).To(MatchError(ContainSubstring("parsing config")))
			})
		})

		DescribeTable("when config file is missing a member",
			func(missingFlag, errorMsg string) {
				allData := map[string]interface{}{
					"poll_interval":                      1234,
					"cni_datastore_path":                 "/some/datastore/path",
					"policy_server_url":                  "https://some-url:1234",
					"vni":                                42,
					"metron_address":                     "http://1.2.3.4:1234",
					"ca_cert_file":                       "/some/ca/file",
					"client_cert_file":                   "/some/client/cert/file",
					"client_key_file":                    "/some/client/key/file",
					"iptables_lock_file":                 "/var/vcap/data/lock",
					"debug_server_host":                  "http://5.6.7.8",
					"debug_server_port":                  5678,
					"log_prefix":                         "cfnetworking",
					"client_timeout_seconds":             5,
					"iptables_accepted_udp_logs_per_sec": 4,
				}
				delete(allData, missingFlag)
				Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())

				_, err = config.New(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("invalid config: %s", errorMsg)))
			},
			Entry("missing poll interval", "poll_interval", "PollInterval: zero value"),
			Entry("missing datastore path", "cni_datastore_path", "Datastore: zero value"),
			Entry("missing policy server url", "policy_server_url", "PolicyServerURL: less than min"),
			Entry("missing vni", "vni", "VNI: zero value"),
			Entry("missing metron address", "metron_address", "MetronAddress: zero value"),
			Entry("missing ca cert", "ca_cert_file", "ServerCACertFile: zero value"),
			Entry("missing client cert file", "client_cert_file", "ClientCertFile: zero value"),
			Entry("missing client key file", "client_key_file", "ClientKeyFile: zero value"),
			Entry("missing iptables lock file", "iptables_lock_file", "IPTablesLockFile: zero value"),
			Entry("missing debug server host", "debug_server_host", "DebugServerHost: zero value"),
			Entry("missing debug server port", "debug_server_port", "DebugServerPort: zero value"),
			Entry("missing log prefix", "log_prefix", "LogPrefix: zero value"),
			Entry("missing client timeout", "client_timeout_seconds", "ClientTimeoutSeconds: zero value"),
			Entry("missing iptables accepted udp logs per sec", "iptables_accepted_udp_logs_per_sec", "IPTablesAcceptedUDPLogsPerSec: less than min"),
		)
	})
})
