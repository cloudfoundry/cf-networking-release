package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"netmon/integration/fakes"
	"os"
	"os/exec"
	"strings"

	"flannel-watchdog/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flannel Watchdog", func() {
	var (
		session        *gexec.Session
		subnetFile     *os.File
		fakeMetron     fakes.FakeMetron
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

	deleteBridge := func() {
		ipLinkSession, err := runCmd("ip", "link", "delete", "dev", bridgeName)
		Expect(err).NotTo(HaveOccurred())
		Eventually(ipLinkSession, DEFAULT_TIMEOUT).Should(gexec.Exit())
	}

	writeSubnetEnv := func() {
		var err error
		subnetFile, err = ioutil.TempFile("", "subnet.env")
		subnetFileName = subnetFile.Name()
		err = ioutil.WriteFile(subnetFileName, []byte(fmt.Sprintf("FLANNEL_SUBNET=%s\nFLANNEL_NETWORK=10.255.0.0/16", bridgeIP)), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())
	}

	startFlannelWatchdog := func() {
		var err error
		configFilePath := WriteConfigFile(config.Config{
			FlannelSubnetFile: subnetFileName,
			BridgeName:        bridgeName,
			MetronAddress:     fakeMetron.Address(),
		})
		watchdogCmd := exec.Command(watchdogBinaryPath, "-config-file", configFilePath)
		session, err = gexec.Start(watchdogCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	}

	Context("when the subnet is the default /24", func() {
		BeforeEach(func() {
			fakeMetron = fakes.New()

			bridgeName = fmt.Sprintf("test-bridge-%d", 100+GinkgoParallelNode())
			bridgeIP = fmt.Sprintf("10.255.%d.1/24", GinkgoParallelNode())

			createBridge()
			writeSubnetEnv()
			startFlannelWatchdog()
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

			deleteBridge()
		})

		It("should log on starting", func() {
			Eventually(session.Out).Should(gbytes.Say("container-networking.flannel-watchdog.*starting"))
		})

		It("should boot and gracefully terminate", func() {
			Consistently(session, "1.5s").ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		It("emits a metric indicating flannel is not down", func() {
			gatherMetricNames := func() map[string]float64 {
				events := fakeMetron.AllEvents()
				metrics := map[string]float64{}
				for _, event := range events {
					metrics[event.Name] = event.Value
				}
				return metrics
			}
			Eventually(gatherMetricNames, "5s").Should(HaveKeyWithValue("flannelDown", 0.0))
			Consistently(gatherMetricNames, "5s").Should(HaveKeyWithValue("flannelDown", 0.0))
		})

		Context("when the subnets file and bridge get out of sync", func() {
			BeforeEach(func() {
				Consistently(session, "1.5s").ShouldNot(gexec.Exit())

				err := ioutil.WriteFile(subnetFileName, []byte(`FLANNEL_SUBNET=10.4.13.1/24\nFLANNEL_NETWORK=10.4.0.0/16`), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			})

			It("exits with a nonzero status", func() {
				expectedMsg := fmt.Sprintf(
					`This cell must be restarted (run \"bosh restart \u003cjob\u003e\").  Flannel is out of sync with the local bridge. `+
						`flannel (%s): 10.4.13.1/24 bridge (%s): %s`, subnetFileName, bridgeName, bridgeIP)
				Expect(string(session.Out.Contents())).To(ContainSubstring(expectedMsg))
			})

			It("emits a metric indicating flannel is down", func() {
				gatherMetricNames := func() map[string]float64 {
					events := fakeMetron.AllEvents()
					metrics := map[string]float64{}
					for _, event := range events {
						metrics[event.Name] = event.Value
					}
					return metrics
				}
				Eventually(gatherMetricNames, "5s").Should(HaveKeyWithValue("flannelDown", 1.0))
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
				Expect(session.Out.Contents()).To(ContainSubstring("open "))
			})
		})

		Context("when the bridge device cannot be found", func() {
			BeforeEach(func() {
				deleteBridge()
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
				Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(
					fmt.Sprintf("Found bridge.*%s", bridgeName)))
				Consistently(howManyFinds, "2s").Should(Equal(1))
			})

			Context("when the bridge is lost and then found", func() {
				It("reports the Found message again", func() {
					Eventually(howManyFinds, DEFAULT_TIMEOUT).Should(Equal(1))

					deleteBridge()

					Eventually(func() string { return string(session.Out.Contents()) }, DEFAULT_TIMEOUT).Should(ContainSubstring("no bridge device found"))

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
				Expect(session.Out).To(gbytes.Say(fmt.Sprintf(
					`container-networking.flannel-watchdog.*device '%s' has no ip`, bridgeName)))
			})
		})
	})

	Context("when the subnet size is set to a /22", func() {
		BeforeEach(func() {
			fakeMetron = fakes.New()

			bridgeName = fmt.Sprintf("test-bridge-%d", 100+GinkgoParallelNode())
			bridgeIP = fmt.Sprintf("10.255.%d.1/22", GinkgoParallelNode())

			createBridge()
			writeSubnetEnv()
			startFlannelWatchdog()
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

			deleteBridge()
		})

		It("should boot and gracefully terminate", func() {
			Consistently(session, "1.5s").ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
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
