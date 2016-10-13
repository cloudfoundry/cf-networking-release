package integration_test

import (
	"cf-pusher/config"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Integration", func() {
	var (
		pathToConfig string
	)

	BeforeEach(func() {
		config := config.Config{
			Applications:   5,
			AppInstances:   2,
			Policies:       4,
			ProxyInstances: 3,
		}

		pathToConfig = WriteConfigFile(config)
	})
	It("takes a config and outs the app configuration", func() {
		session := StartSession(cfPusher, "--config", pathToConfig)
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(session.Out.Contents()).To(MatchJSON(`{
			"org": "scale-org",	
			"space": "scale-space",	
			"tick-apps": ["scale-tick-1", "scale-tick-2", "scale-tick-3","scale-tick-4","scale-tick-5"],
			"tick-instances": 2,
			"registry": "scale-registry",
			"proxy-app": "scale-proxy",
			"proxy-instances": 3
		}`))
	})
})

func StartSession(command string, args ...string) *gexec.Session {
	cmd := exec.Command(command, args...)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}
