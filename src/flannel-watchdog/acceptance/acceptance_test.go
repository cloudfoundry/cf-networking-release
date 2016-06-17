package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

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

	BeforeEach(func() {
		var err error

		bridgeName = fmt.Sprintf("test-bridge-%d", GinkgoParallelNode())
		bridgeIP = "10.255.78.1/24"

		_, err = runCmd("ip", "link", "add", "name", bridgeName, "type", "bridge")
		Expect(err).NotTo(HaveOccurred())

		_, err = runCmd("ip", "addr", "add", bridgeIP, "dev", bridgeName)
		Expect(err).NotTo(HaveOccurred())

		subnetFile, err = ioutil.TempFile("", "subnet.env")
		subnetFileName = subnetFile.Name()
		_, err = subnetFile.WriteString(fmt.Sprintf("FLANNEL_SUBNET=%s", bridgeIP))
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

		_, err := runCmd("ip", "link", "delete", "dev", bridgeName)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should boot and gracefully terminate", func() {
		Consistently(session, "1.5s").ShouldNot(gexec.Exit())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	})

	Context("when the subnets file and bridge get out of sync", func() {
		It("exits with a nonzero status", func() {
			Consistently(session, "1.5s").ShouldNot(gexec.Exit())

			err := ioutil.WriteFile(subnetFileName, []byte("FLANNEL_SUBNET=10.255.13.1/24"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring("out of sync"))
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
			_, err := runCmd("ip", "link", "delete", "dev", bridgeName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("exits with nonzero status code and logs the error", func() {
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring(fmt.Sprintf(`Device "%s" does not exist`, bridgeName)))
		})
	})

	Context("when the bridge device ip is not found", func() {
		BeforeEach(func() {
			_, err := runCmd("ip", "addr", "del", bridgeIP, "dev", bridgeName)
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
