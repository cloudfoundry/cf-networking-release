package integration_test

import (
	"net"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/localip"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"

	"netmon/config"
)

func discoverInterfaceName() string {
	localIP, err := localip.LocalIP()
	Expect(err).NotTo(HaveOccurred())
	ifaces, err := net.Interfaces()
	Expect(err).NotTo(HaveOccurred())
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		Expect(err).NotTo(HaveOccurred())
		for _, addr := range addrs {
			if localIP == strings.Split(addr.String(), "/")[0] {
				return iface.Name
			}
		}
	}
	return ""
}

var _ = Describe("Integration", func() {
	var (
		session    *gexec.Session
		conf       config.Netmon
		fakeMetron metrics.FakeMetron
		ifName     string
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
		configFilePath := WriteConfigFile(conf)

		var err error
		netmonCmd := exec.Command(binaryPath, "-config-file", configFilePath)
		session, err = gexec.Start(netmonCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		runAndWait("ip", "link", "set", "dev", ifName, "mtu", "1500")

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		Expect(fakeMetron.Close()).To(Succeed())
	})

	It("should log when starting", func() {
		Eventually(session.Out).Should(gbytes.Say("cfnetworking.netmon.*starting"))
	})

	It("should emit a metric of the number of network interfaces", func() {
		ifaces, err := net.Interfaces()
		Expect(err).NotTo(HaveOccurred())
		nIfaces := len(ifaces)

		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(metrics.Event{
			EventType: "ValueMetric",
			Name:      "NetInterfaceCount",
			Origin:    "netmon",
			Value:     float64(nIfaces),
		}))
	})

	It("should emit a metric of the total number of iptables rules", func() {
		filterRules := runAndWait("iptables", "-S")
		natRules := runAndWait("iptables", "-S", "-t", "nat")
		totalRulesBaseline := numLines(filterRules) + numLines(natRules)

		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(metrics.Event{
			EventType: "ValueMetric",
			Name:      "IPTablesRuleCount",
			Origin:    "netmon",
			Value:     float64(totalRulesBaseline),
		}))

		runAndWait("iptables", "-w", "-A", "FORWARD", "-s", "1.1.1.1", "-d", "2.2.2.2", "-j", "ACCEPT")

		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(metrics.Event{
			EventType: "ValueMetric",
			Name:      "IPTablesRuleCount",
			Origin:    "netmon",
			Value:     float64(totalRulesBaseline + 1),
		}))
	})

	IsMetricWithName := func(name string) types.GomegaMatcher {
		return WithTransform(func(e metrics.Event) bool {
			return e.Name == name
		}, BeTrue())
	}

	It("should emit metrics for dropped packets", func() {
		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(IsMetricWithName("OverlayRxDropped")))
		Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(IsMetricWithName("OverlayTxDropped")))
	})
})

func runAndWait(bin string, args ...string) string {
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	return string(out)
}

func numLines(text string) int {
	allLines := strings.Split(text, "\n")
	counter := 0
	for _, l := range allLines {
		if len(strings.TrimSpace(l)) > 0 {
			counter++
		}
	}
	return counter
}
