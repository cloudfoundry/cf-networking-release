package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/datastore"
	"net"
	"net/http"
	"os"
	"os/exec"

	"code.cloudfoundry.org/go-db-helpers/metrics"

	"flannel-watchdog/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flannel Watchdog", func() {
	var (
		session          *gexec.Session
		subnetFile       *os.File
		fakeMetron       metrics.FakeMetron
		subnetFileName   string
		metadataFileName string
		cellSubnet       string
		port             int
	)

	writeContainerMetadata := func() {
		containerIP, _, err := net.ParseCIDR(cellSubnet)
		Expect(err).NotTo(HaveOccurred())
		data := map[string]datastore.Container{
			"container-1": datastore.Container{
				Handle: "some-handle",
				IP:     containerIP.String(),
			},
		}

		metadata, err := json.Marshal(data)
		Expect(err).NotTo(HaveOccurred())

		metadataFile, err := ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		metadataFileName = metadataFile.Name()
		err = ioutil.WriteFile(metadataFileName, metadata, os.ModePerm)
		Expect(err).NotTo(HaveOccurred())
	}

	writeSubnetEnv := func() {
		var err error
		subnetFile, err = ioutil.TempFile("", "subnet.env")
		subnetFileName = subnetFile.Name()
		err = ioutil.WriteFile(subnetFileName, []byte(fmt.Sprintf("FLANNEL_SUBNET=%s\nFLANNEL_NETWORK=10.255.0.0/16", cellSubnet)), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())
	}

	startFlannelWatchdog := func() {
		var err error
		port = GinkgoParallelNode() + 4000
		configFilePath := WriteConfigFile(config.Config{
			FlannelSubnetFile: subnetFileName,
			MetronAddress:     fakeMetron.Address(),
			MetadataFilename:  metadataFileName,
			HealthCheckPort:   port,
		})
		watchdogCmd := exec.Command(watchdogBinaryPath, "-config-file", configFilePath)
		session, err = gexec.Start(watchdogCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	}

	Context("when the subnet is the default /24", func() {
		BeforeEach(func() {
			fakeMetron = metrics.NewFakeMetron()
			cellSubnet = fmt.Sprintf("10.255.%d.1/24", GinkgoParallelNode())

			writeContainerMetadata()
			writeSubnetEnv()
			startFlannelWatchdog()
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
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

		It("responds with Status.OK on its health check endpoint", func() {
			client := http.DefaultClient
			callHealthcheck := func() (int, error) {
				resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d", port))
				if resp == nil {
					return -1, err
				}
				return resp.StatusCode, err
			}
			Eventually(callHealthcheck, "5s").Should(Equal(http.StatusOK))
			resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d", port))
			Expect(err).NotTo(HaveOccurred())
			responseBytes, err := ioutil.ReadAll(resp.Body)
			Expect(string(responseBytes)).To(Equal("The cell is healthy! The cell is configured with the correct subnet.\n"))
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

		Context("when the container ips and subnet env ip range are out of sync", func() {
			BeforeEach(func() {
				Consistently(session, "1.5s").ShouldNot(gexec.Exit())

				err := ioutil.WriteFile(subnetFileName, []byte(`FLANNEL_SUBNET=10.4.13.1/24\nFLANNEL_NETWORK=10.4.0.0/16`), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			})

			It("exits with a nonzero status", func() {
				expectedMsg := fmt.Sprintf(
					`This cell must be restarted (run \"bosh restart \u003cjob\u003e\").  Flannel is out of sync with current containers.`)
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

			It("stops responding on its health check endpoint", func() {
				client := http.DefaultClient
				callHealthcheck := func() error {
					_, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d", port))
					return err
				}
				Eventually(callHealthcheck, "5s").ShouldNot(Succeed())
				Consistently(callHealthcheck, "2s").ShouldNot(Succeed())
			})

		})
	})

	Context("when the subnet size is set to a /22", func() {
		BeforeEach(func() {
			fakeMetron = metrics.NewFakeMetron()
			cellSubnet = fmt.Sprintf("10.255.%d.1/22", GinkgoParallelNode())

			writeContainerMetadata()
			writeSubnetEnv()
			startFlannelWatchdog()
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
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
