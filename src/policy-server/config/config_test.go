package config_test

import (
	"io/ioutil"
	"os"
	"policy-server/config"

	. "github.com/onsi/ginkgo"
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
					"internal_listen_port": 2222,
					"ca_cert_file": "some/ca/cert/file",
					"server_cert_file": "some/server/cert/file",
					"server_key_file": "some/server/key/file",
					"uaa_client": "some-uaa-client",
					"uaa_client_secret": "some-uaa-client-secret",
					"uaa_url": "http://uaa.example.com",
					"cc_url": "http://ccapi.example.com",
					"skip_ssl_validation": true,
					"database": {
						"type": "mysql",
						"connection_string": "some-db-connection-string"
					},
					"tag_length": 2,
					"metron_address": "http://1.2.3.4:9999"
				}`)
				c, err := config.New(file.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(c.ListenHost).To(Equal("http://1.2.3.4"))
				Expect(c.ListenPort).To(Equal(1234))
				Expect(c.InternalListenPort).To(Equal(2222))
				Expect(c.CACertFile).To(Equal("some/ca/cert/file"))
				Expect(c.ServerCertFile).To(Equal("some/server/cert/file"))
				Expect(c.ServerKeyFile).To(Equal("some/server/key/file"))
				Expect(c.UAAClient).To(Equal("some-uaa-client"))
				Expect(c.UAAClientSecret).To(Equal("some-uaa-client-secret"))
				Expect(c.UAAURL).To(Equal("http://uaa.example.com"))
				Expect(c.CCURL).To(Equal("http://ccapi.example.com"))
				Expect(c.SkipSSLValidation).To(Equal(true))
				Expect(c.Database.Type).To(Equal("mysql"))
				Expect(c.Database.ConnectionString).To(Equal("some-db-connection-string"))
				Expect(c.TagLength).To(Equal(2))
				Expect(c.MetronAddress).To(Equal("http://1.2.3.4:9999"))
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
	})
})
