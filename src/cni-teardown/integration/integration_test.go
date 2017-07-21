package integration_test

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"os/exec"

	"code.cloudfoundry.org/silk/cni/config"
	"code.cloudfoundry.org/silk/cni/lib"
	"code.cloudfoundry.org/silk/lib/adapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/vishvananda/netlink"
)

var (
	DEFAULT_TIMEOUT = "5s"

	ifbConfig      config.Config
	ifbCreator     *lib.IFBCreator
	netlinkAdapter *adapter.NetlinkAdapter
	ifbName        string
)

var _ = BeforeEach(func() {
	ifbName = "i-some-ifb"

	ifbConfig = config.Config{}
	ifbConfig.Container.MTU = 1350
	ifbConfig.IFB.DeviceName = ifbName

	netlinkAdapter = &adapter.NetlinkAdapter{}
	ifbCreator = &lib.IFBCreator{
		NetlinkAdapter:      netlinkAdapter,
		DeviceNameGenerator: &config.DeviceNameGenerator{},
	}
	Expect(netlink.LinkAdd(&netlink.Ifb{
		LinkAttrs: netlink.LinkAttrs{
			Name:  ifbConfig.IFB.DeviceName,
			Flags: net.FlagUp,
			MTU:   ifbConfig.Container.MTU,
		}})).To(Succeed())
	// Expect(ifbCreator.Create(&ifbConfig)).To(Succeed())
})

var _ = AfterEach(func() {
	removeIFB(ifbConfig)
})

var _ = Describe("Teardown", func() {
	It("destroys the IFB device", func() {
		By("running teardown")
		session := runTeardown()
		Expect(session).To(gexec.Exit(0))

		By("verifying that the ifb is no longer present")
		_, err := netlinkAdapter.LinkByName(ifbName)
		Expect(err).To(MatchError("find link: Link not found"))

		Expect(session.Out.Contents()).To(ContainSubstring("cni-teardown.complete"))
	})
})

func removeIFB(ifbConfig config.Config) {
	exec.Command("ip", "link", "del", ifbConfig.IFB.DeviceName).Run()
}

func writeConfigFile(config config.Config) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}

func runTeardown() *gexec.Session {
	startCmd := exec.Command(paths.TeardownBin)
	// startCmd := exec.Command(paths.TeardownBin, "--config", configFilePath)
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	return session
}
