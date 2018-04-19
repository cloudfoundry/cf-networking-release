package lib_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"proxy-plugin/lib"
)

var _ = Describe("ProxyConfig", func() {
	Describe("LoadProxyConfig", func() {
		It("should parse a valid config", func() {
			input := []byte(`{
				"proxy_port": 6868,
				"proxy_range": "10.255.0.0/16"
			}`)
			result, err := lib.LoadProxyConfig(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&lib.ProxyConfig{
				ProxyPort:  6868,
				ProxyRange: "10.255.0.0/16",
			}))
		})

		It("should return an error when parsing an invalid json", func() {
			input := []byte(`(>'-')> <('-'<) ^(' - ')^ <('-'<) (>'-')>`)
			_, err := lib.LoadProxyConfig(input)
			Expect(err.Error()).To(HavePrefix("loading proxy config: "))
		})
	})
})
