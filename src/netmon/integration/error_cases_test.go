package integration_test

import (
	"netmon/config"
	"os/exec"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Integration", func() {
	var (
		session        *gexec.Session
		conf           config.Netmon
		fakeMetron     metrics.FakeMetron
		ifName         string
		configFilePath string
	)
	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		ifName = discoverInterfaceName()
		conf = config.Netmon{
			PollInterval:  1,
			MetronAddress: fakeMetron.Address(),
			InterfaceName: ifName,
			LogLevel:      "info",
			LogPrefix:     "cfnetworking",
		}
	})
	Context("when the config file is invalid", func() {
		BeforeEach(func() {
			conf.InterfaceName = ""
			configFilePath = WriteConfigFile(conf)
			var err error
			netmonCmd := exec.Command(binaryPath, "-config-file", configFilePath)
			session, err = gexec.Start(netmonCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})
		It("logs the error and exits 1", func() {
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			Expect(string(session.Err.Contents())).To(ContainSubstring("cfnetworking.netmon: reading config: invalid config: InterfaceName: zero value"))
		})
	})
	Context("when the config file argument is not included", func() {
		BeforeEach(func() {
			var err error
			netmonCmd := exec.Command(binaryPath)
			session, err = gexec.Start(netmonCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})
		It("logs the error and exits 1", func() {
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			Expect(string(session.Err.Contents())).To(ContainSubstring("cfnetworking.netmon: reading config: file does not exist: stat : no such file or directory"))
		})
	})
})
