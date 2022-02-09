package config_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/policy-server/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("ASGSyncerConfig", func() {
	Describe("NewASGSyncer", func() {
		var (
			validConfig map[string]interface{}
			file        *os.File
			err         error
		)

		BeforeEach(func() {
			validConfig = map[string]interface{}{
				"asg_poll_interval_seconds": 60,
				"uuid":                      "some-uuid",
				"database": map[string]interface{}{
					"type":          "mysql",
					"user":          "root",
					"password":      "password",
					"host":          "127.0.0.1",
					"port":          3306,
					"timeout":       5,
					"database_name": "network_policy",
				},
				"uaa_client":          "some-uaa-client",
				"uaa_client_secret":   "some-uaa-client-secret",
				"uaa_ca":              "some/ca/cert/uaa.ca",
				"uaa_url":             "uaa.service.cf.internal",
				"uaa_port":            8443,
				"cc_url":              "cc.service.cf.internal",
				"cc_ca_cert":          "some/ca/cert/cc.ca",
				"log_prefix":          "cfnetworking",
				"log_level":           "debug",
				"metron_address":      "127.0.0.1:3457",
				"skip_ssl_validation": true,
				"locket": map[string]interface{}{
					"locket_address":          "http://6.5.4.3",
					"locket_ca_cert_file":     "some/ca/cert/locket.ca",
					"locket_client_cert_file": "some/client/cert/locket.cert",
					"locket_client_key_file":  "some/client/cert/locket.key",
				},
			}
			file, err = ioutil.TempFile(os.TempDir(), "config-")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the config file is valid", func() {
			It("returns the config", func() {
				Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				c, err := config.NewASGSyncer(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.ASGSyncInterval).To(Equal(60))
				Expect(c.UUID).To(Equal("some-uuid"))
				Expect(c.Database.Type).To(Equal("mysql"))
				Expect(c.Database.User).To(Equal("root"))
				Expect(c.Database.Password).To(Equal("password"))
				Expect(c.Database.Host).To(Equal("127.0.0.1"))
				Expect(c.Database.Port).To(Equal(uint16(3306)))
				Expect(c.Database.Timeout).To(Equal(5))
				Expect(c.Database.DatabaseName).To(Equal("network_policy"))
				Expect(c.UAAClient).To(Equal("some-uaa-client"))
				Expect(c.UAAClientSecret).To(Equal("some-uaa-client-secret"))
				Expect(c.UAACA).To(Equal("some/ca/cert/uaa.ca"))
				Expect(c.UAAURL).To(Equal("uaa.service.cf.internal"))
				Expect(c.UAAPort).To(Equal(8443))
				Expect(c.CCURL).To(Equal("cc.service.cf.internal"))
				Expect(c.CCCA).To(Equal("some/ca/cert/cc.ca"))
				Expect(c.LogPrefix).To(Equal("cfnetworking"))
				Expect(c.LogLevel).To(Equal("debug"))
				Expect(c.MetronAddress).To(Equal("127.0.0.1:3457"))
				Expect(c.SkipSSLValidation).To(Equal(true))
			})
		})

		Context("when the config file path does not exist", func() {
			It("returns a meaningful error", func() {
				_, err := config.NewASGSyncer("/some/bad/filepath")
				Expect(err).To(MatchError(HavePrefix("reading config: open /some/bad/filepath:")))
			})
		})

		Context("when config file contents are blank", func() {
			It("returns the error", func() {
				_, err = config.NewASGSyncer(file.Name())
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

				_, err = config.NewASGSyncer(configFile.Name())
				Expect(err).To(MatchError("parsing config: unexpected end of JSON input"))
			})
		})

		DescribeTable("when config file is missing a member",
			func(missingFlag, errorMsg string) {
				delete(validConfig, missingFlag)
				Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())

				_, err = config.NewASGSyncer(file.Name())
				Expect(err).To(MatchError(fmt.Sprintf("invalid config: %s", errorMsg)))
			},
			Entry("missing log prefix", "log_prefix", "LogPrefix: zero value"),
			Entry("missing uaa client", "uaa_client", "UAAClient: zero value"),
			Entry("missing uaa client secret", "uaa_client_secret", "UAAClientSecret: zero value"),
			Entry("missing uaa url", "uaa_url", "UAAURL: zero value"),
			Entry("missing uaa port", "uaa_port", "UAAPort: zero value"),
			Entry("missing cc url", "cc_url", "CCURL: zero value"),
		)

		Describe("asg sync interval", func() {
			Context("when the asg sync interval is less than 0", func() {
				BeforeEach(func() {
					validConfig["asg_poll_interval_seconds"] = -10
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).To(MatchError("invalid config: ASGSyncInterval: less than min"))
				})
			})
		})

		Describe("database config", func() {
			Context("when the config file is missing a db type", func() {
				BeforeEach(func() {
					delete(validConfig["database"].(map[string]interface{}), "type")
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Type: zero value"))
				})
			})

			Context("when the config file is missing a user", func() {
				BeforeEach(func() {
					delete(validConfig["database"].(map[string]interface{}), "user")
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})

				It("returns an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).To(MatchError("invalid config: Database.User: zero value"))
				})
			})

			Context("when the config file is missing a password", func() {
				BeforeEach(func() {
					delete(validConfig["database"].(map[string]interface{}), "password")
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})

				It("does not return an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the config file is missing a host", func() {
				BeforeEach(func() {
					delete(validConfig["database"].(map[string]interface{}), "host")
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})

				It("returns an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Host: zero value"))
				})
			})

			Context("when the config file is missing a port", func() {
				BeforeEach(func() {
					delete(validConfig["database"].(map[string]interface{}), "port")
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})

				It("returns an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Port: zero value"))
				})
			})

			Context("when the config file is missing a timeout", func() {
				BeforeEach(func() {
					delete(validConfig["database"].(map[string]interface{}), "timeout")
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})
				It("returns an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).To(MatchError("invalid config: Database.Timeout: less than min"))
				})
			})

			Context("when the config file is missing a database_name", func() {
				BeforeEach(func() {
					delete(validConfig["database"].(map[string]interface{}), "database_name")
					Expect(json.NewEncoder(file).Encode(validConfig)).To(Succeed())
				})
				It("does not return an error", func() {
					_, err = config.NewASGSyncer(file.Name())
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})
