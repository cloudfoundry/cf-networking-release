package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"flannel-watchdog/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flannel Watchdog", func() {
	var (
		session        *gexec.Session
		subnetFile     *os.File
		subnetFileName string
		bridgeName     string
		bridgeIP       string
	)

	createBridge := func() {
		ipLinkSession, err := runCmd("ip", "link", "add", "name", bridgeName, "type", "bridge")
		Expect(err).NotTo(HaveOccurred())
		Eventually(ipLinkSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		ipLinkSession, err = runCmd("ip", "addr", "add", bridgeIP, "dev", bridgeName)
		Expect(err).NotTo(HaveOccurred())
		Eventually(ipLinkSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	}

	deleteBridge := func(allowError bool) {
		ipLinkSession, err := runCmd("ip", "link", "delete", "dev", bridgeName)
		Expect(err).NotTo(HaveOccurred())
		if allowError {
			Eventually(ipLinkSession, DEFAULT_TIMEOUT).Should(gexec.Exit())
		} else {
			Eventually(ipLinkSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		}
	}

	BeforeEach(func() {
		bridgeName = fmt.Sprintf("test-bridge-%d", 100+GinkgoParallelNode())
		bridgeIP = fmt.Sprintf("10.255.%d.1/24", GinkgoParallelNode())

		createBridge()

		var err error
		subnetFile, err = ioutil.TempFile("", "subnet.env")
		subnetFileName = subnetFile.Name()
		err = ioutil.WriteFile(subnetFileName, []byte(fmt.Sprintf("FLANNEL_SUBNET=%s\nFLANNEL_NETWORK=10.255.0.0/16", bridgeIP)), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		configFilePath := WriteConfigFile(config.Config{
			FlannelSubnetFile: subnetFileName,
			BridgeName:        bridgeName,
		})
		watchdogCmd := exec.Command(watchdogBinaryPath, "-config-file", configFilePath)
		session, err = gexec.Start(watchdogCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		deleteBridge(true)
	})

	It("should boot and gracefully terminate", func() {
		Consistently(session, "1.5s").ShouldNot(gexec.Exit())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	})

	Context("when the subnets file and bridge get out of sync", func() {
		It("exits with a nonzero status", func() {
			Consistently(session, "1.5s").ShouldNot(gexec.Exit())

			err := ioutil.WriteFile(subnetFileName, []byte(`FLANNEL_SUBNET=10.4.13.1/24\nFLANNEL_NETWORK=10.4.0.0/16`), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			expectedMsg := fmt.Sprintf(
				`This cell must be recreated.  Flannel is out of sync with the local bridge. `+
					`flannel (%s): 10.4.13.1/24 bridge (%s): %s`, subnetFileName, bridgeName, bridgeIP)
			Expect(string(session.Err.Contents())).To(ContainSubstring(expectedMsg))
		})
	})

	Context("when the flannel env file cannot be read", func() {
		BeforeEach(func() {
			var err error
			err = os.Remove(subnetFileName)
			Expect(err).NotTo(HaveOccurred())
		})
		It("exits with nonzero status code and logs the error", func() {
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring("open "))
		})
	})

	Context("when the bridge device cannot be found", func() {
		BeforeEach(func() {
			deleteBridge(false)
		})

		It("continues running", func() {
			Consistently(session, "1.5s").ShouldNot(gexec.Exit())
		})
	})

	Context("once the bridge device is found", func() {
		howManyFinds := func() int {
			return strings.Count(string(session.Out.Contents()), "Found bridge")
		}

		It("reports this fact and then shuts up", func() {
			expectedMsg := fmt.Sprintf("Found bridge %s", bridgeName)
			Eventually(session.Out.Contents, DEFAULT_TIMEOUT).Should(ContainSubstring(expectedMsg))
			Consistently(howManyFinds, "2s").Should(Equal(1))
		})

		Context("when the bridge is lost and then found", func() {
			It("reports the Found message again", func() {
				Eventually(howManyFinds, DEFAULT_TIMEOUT).Should(Equal(1))

				deleteBridge(false)

				Eventually(session.Out.Contents, DEFAULT_TIMEOUT).Should(ContainSubstring("no bridge device found"))

				createBridge()

				Eventually(howManyFinds, DEFAULT_TIMEOUT).Should(Equal(2))
			})
		})
	})

	Context("when the bridge device ip is not found", func() {
		BeforeEach(func() {
			ipDelSession, err := runCmd("ip", "addr", "del", bridgeIP, "dev", bridgeName)
			Eventually(ipDelSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
			Expect(err).NotTo(HaveOccurred())
		})

		It("exits with nonzero status code and logs the error", func() {
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring(fmt.Sprintf(`device "%s" has no ip`, bridgeName)))
		})
	})

})

func WriteConfigFile(conf config.Config) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(conf)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}

func runCmd(command string, args ...string) (*gexec.Session, error) {
	cmd := exec.Command(command, args...)
	return gexec.Start(cmd, nil, nil)
}
