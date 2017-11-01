package version_test

import (
	"cli-plugin/version"

	"code.cloudfoundry.org/cli/plugin"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Getter", func() {
	var (
		getter *version.Getter
	)

	BeforeEach(func() {
		getter = &version.Getter{}
	})

	Describe("Get", func() {
		It("gets the current version", func() {
			Expect(getter.Get()).To(Equal(plugin.VersionType{
				Major: 1,
				Minor: 6,
				Build: 1,
			}))
		})
	})
})
