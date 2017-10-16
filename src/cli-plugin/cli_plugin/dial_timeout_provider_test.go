package cli_plugin

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DialTimeoutProvider", func() {
	var provider dialTimeoutProvider

	BeforeEach(func() {
		provider = dialTimeoutProvider{}
	})

	It("returns 3 seconds by default", func() {
		Expect(provider.Get()).To(Equal(3 * time.Second))
	})

	Context("when the environment defines the CF_DIAL_TIMEOUT env var", func() {
		var (
			oldDialTimeout string
			hasOriginalEnv bool
		)

		BeforeEach(func() {
			oldDialTimeout, hasOriginalEnv = os.LookupEnv("CF_DIAL_TIMEOUT")
			os.Setenv("CF_DIAL_TIMEOUT", "100")
		})

		AfterEach(func() {
			if hasOriginalEnv {
				os.Setenv("CF_DIAL_TIMEOUT", oldDialTimeout)
			} else {
				os.Unsetenv("CF_DIAL_TIMEOUT")
			}
		})

		It("returns value from env (100) seconds", func() {
			Expect(provider.Get()).To(Equal(100 * time.Second))
		})

		Context("when the env var is garbage", func() {
			BeforeEach(func() {
				os.Setenv("CF_DIAL_TIMEOUT", "potato")
			})


			It("returns the default 3 seconds", func() {
				Expect(provider.Get()).To(Equal(3 * time.Second))
			})
		})
	})
})
