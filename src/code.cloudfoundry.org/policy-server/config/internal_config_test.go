package config_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/policy-server/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("InternalConfig", func() {
	Describe("NewInternal", func() {
		var (
			file *os.File
			err  error
		)

		BeforeEach(func() {
			file, err = ioutil.TempFile(os.TempDir(), "config-")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the config file is valid", func() {
			It("returns the config", func() {
				file.WriteString(`{
					"log_prefix": "cfnetworking",
					"listen_host": "http://1.2.3.4",
					"internal_listen_port": 2222,
					"debug_server_host": "http://6.5.4.3",
					"debug_server_port": 9999,
					"health_check_port": 9443,
					"ca_cert_file": "some/ca/cert/file",
					"server_cert_file": "some/server/cert/file",
					"server_key_file": "some/server/key/file",
					"database": {
						"type": "mysql",
						"user": "root",
						"password": "password",
						"host": "127.0.0.1",
						"port": 3306,
						"timeout": 5,
						"database_name": "network_policy"
					},
					"max_idle_connections": 4,
					"max_open_connections": 5,
					"connections_max_lifetime_seconds": 45,
					"tag_length": 2,
					"metron_address": "http://1.2.3.4:9999",
					"log_level": "debug"
				}`)
				c, err := config.NewInternal(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.LogPrefix).To(Equal("cfnetworking"))
				Expect(c.ListenHost).To(Equal("http://1.2.3.4"))
				Expect(c.InternalListenPort).To(Equal(2222))
				Expect(c.DebugServerHost).To(Equal("http://6.5.4.3"))
				Expect(c.DebugServerPort).To(Equal(9999))
				Expect(c.HealthCheckPort).To(Equal(9443))
				Expect(c.CACertFile).To(Equal("some/ca/cert/file"))
				Expect(c.ServerCertFile).To(Equal("some/server/cert/file"))
				Expect(c.ServerKeyFile).To(Equal("some/server/key/file"))
				Expect(c.Database.Type).To(Equal("mysql"))
				Expect(c.Database.User).To(Equal("root"))
				Expect(c.Database.Password).To(Equal("password"))
				Expect(c.Database.Host).To(Equal("127.0.0.1"))
				Expect(c.Database.Port).To(Equal(uint16(3306)))
				Expect(c.Database.Timeout).To(Equal(5))
				Expect(c.Database.DatabaseName).To(Equal("network_policy"))
				Expect(c.TagLength).To(Equal(2))
				Expect(c.MetronAddress).To(Equal("http://1.2.3.4:9999"))
				Expect(c.LogLevel).To(Equal("debug"))
				Expect(c.MaxIdleConnections).To(Equal(4))
				Expect(c.MaxOpenConnections).To(Equal(5))
				Expect(c.MaxConnectionsLifetimeSeconds).To(Equal(45))
			})
		})

		Context("when the config file path does not exist", func() {
			It("returns a meaningful error", func() {
				_, err := config.NewInternal("/some/bad/filepath")
				Expect(err).To(MatchError(HavePrefix("reading config: open /some/bad/filepath:")))
			})
		})

		Context("when config file contents are blank", func() {
			It("returns the error", func() {
				_, err = config.NewInternal(file.Name())
				Expect(err).To(MatchError(ContainSubstring("parsing config")))
			})
		})

		Context("when the file has invalid json", func() {
			It("returns a meaningful error", func() {
				configFile, err := ioutil.TempFile("", "config")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(configFile.Name())

				_, err = configFile.WriteString(`{"listen_host":"some.host.name"`)
				Expect(err).NotTo(HaveOccurred())
				Expect(configFile.Close()).To(Succeed())

				_, err = config.NewInternal(configFile.Name())
				Expect(err).To(MatchError("parsing config: unexpected end of JSON input"))
			})
		})

		DescribeTable("when config file is missing a member",
			func(missingFlag, errorMsg string) {
				allData := map[string]interface{}{
					"log_prefix":           "cfnetworking",
					"listen_host":          "http://1.2.3.4",
					"internal_listen_port": 2222,
					"debug_server_host":    "http://4.4.4.4",
					"debug_server_port":    3333,
					"health_check_port":    4444,
					"ca_cert_file":         "some/ca/cert/file",
					"server_cert_file":     "some/server/cert/file",
					"server_key_file":      "some/server/key/file",
					"database": map[string]interface{}{
						"type":          "mysql",
						"user":          "root",
						"password":      "password",
						"host":          "127.0.0.1",
						"port":          3306,
						"timeout":       5,
						"database_name": "network_policy",
					},
					"tag_length":     2,
					"metron_address": "http://1.2.3.4:9999",
				}
				delete(allData, missingFlag)
				Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())

				_, err = config.NewInternal(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("invalid config: %s", errorMsg)))
			},
			Entry("missing log prefix", "log_prefix", "LogPrefix: zero value"),
			Entry("missing listen host", "listen_host", "ListenHost: zero value"),
			Entry("missing internal listen port", "internal_listen_port", "InternalListenPort: zero value"),
			Entry("missing debug server host", "debug_server_host", "DebugServerHost: zero value"),
			Entry("missing debug server port", "debug_server_port", "DebugServerPort: zero value"),
			Entry("missing health check port", "health_check_port", "HealthCheckPort: zero value"),
			Entry("missing ca cert file", "ca_cert_file", "CACertFile: zero value"),
			Entry("missing server cert file", "server_cert_file", "ServerCertFile: zero value"),
			Entry("missing server key file", "server_key_file", "ServerKeyFile: zero value"),
			Entry("missing tag length", "tag_length", "TagLength: zero value"),
			Entry("missing metron address", "metron_address", "MetronAddress: zero value"),
		)

		Describe("database config", func() {
			var allData map[string]interface{}
			BeforeEach(func() {
				allData = map[string]interface{}{
					"listen_host":          "http://1.2.3.4",
					"listen_port":          1234,
					"log_prefix":           "cfnetworking",
					"internal_listen_port": 2222,
					"debug_server_host":    "http://4.4.4.4",
					"debug_server_port":    3333,
					"health_check_port":    3333,
					"ca_cert_file":         "some/ca/cert/file",
					"server_cert_file":     "some/server/cert/file",
					"server_key_file":      "some/server/key/file",
					"uaa_client":           "some-uaa-client",
					"uaa_client_secret":    "some-uaa-client-secret",
					"uaa_url":              "http://uaa.example.com",
					"uaa_port":             7777,
					"cc_url":               "http://ccapi.example.com",
					"skip_ssl_validation":  true,
					"database": map[string]interface{}{
						"type":          "mysql",
						"user":          "root",
						"password":      "password",
						"host":          "127.0.0.1",
						"port":          3306,
						"timeout":       5,
						"database_name": "network_policy",
					},
					"tag_length":       2,
					"metron_address":   "http://1.2.3.4:9999",
					"log_level":        "info",
					"cleanup_interval": 2,
					"max_policies":     3,
				}
			})

			Context("when the config file is missing a db type", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "type")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Type: zero value"))
				})
			})

			Context("when the config file is missing a user", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "user")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: Database.User: zero value"))
				})
			})

			Context("when the config file is missing a password", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "password")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("does not return an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the config file is missing a host", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "host")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Host: zero value"))
				})
			})

			Context("when the config file is missing a port", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "port")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Port: zero value"))
				})
			})

			Context("when the config file is missing a timeout", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "timeout")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Timeout: less than min"))
				})
			})

			Context("when the max idle connections is less than 0", func() {
				BeforeEach(func() {
					allData["max_idle_connections"] = -1
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})

				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: MaxIdleConnections: less than min"))
				})
			})

			Context("when the max open connections is less than 0", func() {
				BeforeEach(func() {
					allData["max_open_connections"] = -1
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})

				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: MaxOpenConnections: less than min"))
				})
			})

			Context("when the connections max lifetime is less than 0", func() {
				BeforeEach(func() {
					allData["connections_max_lifetime_seconds"] = -1
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})

				It("returns an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).To(MatchError("invalid config: MaxConnectionsLifetimeSeconds: less than min"))
				})
			})

			Context("when the config file is missing a database_name", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "database_name")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("does not return an error", func() {
					_, err = config.NewInternal(file.Name())
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})
