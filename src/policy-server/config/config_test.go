package config_test

import (
	"io/ioutil"
	"os"
	"policy-server/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("Load", func() {
		It("loads the config from a file", func() {
			configFile, err := ioutil.TempFile("", "config")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(configFile.Name())

			_, err = configFile.WriteString(`{"listen_host":"some.host.name"}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(configFile.Close()).To(Succeed())

			cfg, err := config.Load(configFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).To(Equal(&config.Config{
				ListenHost: "some.host.name",
			}))
		})

		Context("when the file cannot be read", func() {
			It("returns a meaningful error", func() {
				_, err := config.Load("/some/bad/filepath")
				Expect(err).To(MatchError(HavePrefix("reading config: open /some/bad/filepath:")))
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

				_, err = config.Load(configFile.Name())
				Expect(err).To(MatchError("parsing json: unexpected end of JSON input"))
			})
		})
	})
})
