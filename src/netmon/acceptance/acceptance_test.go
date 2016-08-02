package acceptance_test

import (
	"net"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"netmon/acceptance/fakes"
	"netmon/config"
)

var _ = Describe("Acceptance", func() {
	var (
		session    *gexec.Session
		conf       config.Netmon
		fakeMetron fakes.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = fakes.New()

		conf = config.Netmon{
			PollInterval:  1,
			MetronAddress: fakeMetron.Address(),
		}
		configFilePath := WriteConfigFile(conf)

		netmonCmd := exec.Command(binaryPath, "-config-file", configFilePath)
		var err error
		session, err = gexec.Start(netmonCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		Expect(fakeMetron.Close()).To(Succeed())
	})

	It("should emit a metric of the number of network interfaces", func() {
		ifaces, err := net.Interfaces()
		Expect(err).NotTo(HaveOccurred())
		nIfaces := len(ifaces)

		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(fakes.Event{
			EventType: "ValueMetric",
			Name:      "NetInterfaceCount",
			Origin:    "netmon",
			Value:     float64(nIfaces),
		}))
	})
})
