package config_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"policy-server/config"

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

		Context("when the config file is valid", func() {
			It("returns the config", func() {
				file.WriteString(`{
					"listen_host": "http://1.2.3.4",
					"listen_port": 1234,
					"log_prefix": "cfnetworking",
					"internal_listen_port": 2222,
					"debug_server_host": "http://6.5.4.3",
					"debug_server_port": 9999,
					"uaa_client": "some-uaa-client",
					"uaa_client_secret": "some-uaa-client-secret",
					"uaa_url": "http://uaa.example.com",
					"uaa_port": 8888,
					"uaa_ca": "some/uaa/ca/file",
					"cc_url": "http://ccapi.example.com",
					"skip_ssl_validation": true,
					"database": {
						"type": "mysql",
						"user": "root",
						"password": "password",
						"host": "127.0.0.1",
						"port": 3306,
						"timeout": 5,
						"database_name": "network_policy",
						"require_ssl": true,
						"ca_cert": "/some/ca/cert/path"
					},
					"tag_length": 2,
					"metron_address": "http://1.2.3.4:9999",
					"log_level": "debug",
					"cleanup_interval": 2,
					"request_timeout": 5,
					"max_policies": 3,
					"enable_space_developer_self_service": true,
					"allowed_cors_domains": ["https://foo.bar", "https://bar.foo"]
				}`)
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.ListenHost).To(Equal("http://1.2.3.4"))
				Expect(c.ListenPort).To(Equal(1234))
				Expect(c.LogPrefix).To(Equal("cfnetworking"))
				Expect(c.DebugServerHost).To(Equal("http://6.5.4.3"))
				Expect(c.DebugServerPort).To(Equal(9999))
				Expect(c.UAAClient).To(Equal("some-uaa-client"))
				Expect(c.UAAClientSecret).To(Equal("some-uaa-client-secret"))
				Expect(c.UAAURL).To(Equal("http://uaa.example.com"))
				Expect(c.UAAPort).To(Equal(8888))
				Expect(c.UAACA).To(Equal("some/uaa/ca/file"))
				Expect(c.CCURL).To(Equal("http://ccapi.example.com"))
				Expect(c.SkipSSLValidation).To(Equal(true))
				Expect(c.Database.Type).To(Equal("mysql"))
				Expect(c.Database.User).To(Equal("root"))
				Expect(c.Database.Password).To(Equal("password"))
				Expect(c.Database.Host).To(Equal("127.0.0.1"))
				Expect(c.Database.Port).To(Equal(uint16(3306)))
				Expect(c.Database.Timeout).To(Equal(5))
				Expect(c.Database.DatabaseName).To(Equal("network_policy"))
				Expect(c.Database.RequireSSL).To(Equal(true))
				Expect(c.Database.CACert).To(Equal("/some/ca/cert/path"))
				Expect(c.TagLength).To(Equal(2))
				Expect(c.MetronAddress).To(Equal("http://1.2.3.4:9999"))
				Expect(c.LogLevel).To(Equal("debug"))
				Expect(c.CleanupInterval).To(Equal(2))
				Expect(c.RequestTimeout).To(Equal(5))
				Expect(c.MaxPolicies).To(Equal(3))
				Expect(c.EnableSpaceDeveloperSelfService).To(BeTrue())
				Expect(c.AllowedCORSDomains).To(Equal([]string{
					"https://foo.bar",
					"https://bar.foo",
				}))
			})
		})

		Context("when the config file path does not exist", func() {
			It("returns a meaningful error", func() {
				_, err := config.New("/some/bad/filepath")
				Expect(err).To(MatchError(HavePrefix("reading config: open /some/bad/filepath:")))
			})
		})

		Context("when config file contents are blank", func() {
			It("returns the error", func() {
				_, err = config.New(file.Name())
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

				_, err = config.New(configFile.Name())
				Expect(err).To(MatchError("parsing config: unexpected end of JSON input"))
			})
		})

		DescribeTable("when config file is missing a member",
			func(missingFlag, errorMsg string) {
				allData := map[string]interface{}{
					"listen_host":         "http://1.2.3.4",
					"listen_port":         1234,
					"log_prefix":          "cfnetworking",
					"debug_server_host":   "http://4.4.4.4",
					"debug_server_port":   3333,
					"uaa_client":          "some-uaa-client",
					"uaa_client_secret":   "some-uaa-client-secret",
					"uaa_url":             "http://uaa.example.com",
					"uaa_port":            5555,
					"cc_url":              "http://ccapi.example.com",
					"skip_ssl_validation": true,
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
					"cleanup_interval": 2,
					"request_timeout":  5,
					"max_policies":     3,
				}
				delete(allData, missingFlag)
				Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())

				_, err = config.New(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("invalid config: %s", errorMsg)))
			},
			Entry("missing listen host", "listen_host", "ListenHost: zero value"),
			Entry("missing listen port", "listen_port", "ListenPort: zero value"),
			Entry("missing log prefix", "log_prefix", "LogPrefix: zero value"),
			Entry("missing debug server host", "debug_server_host", "DebugServerHost: zero value"),
			Entry("missing debug server port", "debug_server_port", "DebugServerPort: zero value"),
			Entry("missing uaa client", "uaa_client", "UAAClient: zero value"),
			Entry("missing uaa client secret", "uaa_client_secret", "UAAClientSecret: zero value"),
			Entry("missing uaa url", "uaa_url", "UAAURL: zero value"),
			Entry("missing uaa port", "uaa_port", "UAAPort: zero value"),
			Entry("missing cc url", "cc_url", "CCURL: zero value"),
			Entry("missing tag length", "tag_length", "TagLength: zero value"),
			Entry("missing metron address", "metron_address", "MetronAddress: zero value"),
			Entry("missing cleanup interval", "cleanup_interval", "CleanupInterval: less than min"),
			Entry("missing request timeout", "request_timeout", "RequestTimeout: less than min"),
			Entry("missing max policies", "max_policies", "MaxPolicies: less than min"),
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
					"request_timeout":  5,
					"max_policies":     3,
				}
			})

			Context("when the config file is missing a db type", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "type")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.New(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Type: zero value"))
				})
			})

			Context("when the config file is missing a user", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "user")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.New(file.Name())
					Expect(err).To(MatchError("invalid config: Database.User: zero value"))
				})
			})

			Context("when the config file is missing a password", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "password")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("does not return an error", func() {
					_, err = config.New(file.Name())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the config file is missing a host", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "host")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.New(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Host: zero value"))
				})
			})

			Context("when the config file is missing a port", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "port")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.New(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Port: zero value"))
				})
			})

			Context("when the config file is missing a timeout", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "timeout")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.New(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Timeout: less than min"))
				})
			})

			Context("when the config file is missing a database_name", func() {
				BeforeEach(func() {
					delete(allData["database"].(map[string]interface{}), "database_name")
					Expect(json.NewEncoder(file).Encode(allData)).To(Succeed())
				})
				It("does not return an error", func() {
					_, err = config.New(file.Name())
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})
