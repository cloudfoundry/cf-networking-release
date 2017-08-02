package integration_test

import (
	"cni-wrapper-plugin/lib"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"strings"
	"time"

	"code.cloudfoundry.org/silk/lib/adapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	DEFAULT_TIMEOUT = "5s"

	config                *lib.WrapperConfig
	netlinkAdapter        *adapter.NetlinkAdapter
	ifbName               string
	notSilkCreatedIFBName string
	dummyName             string
	configFilePath        string
	datastorePath         string
	delegateDataDirPath   string
	delegateDatastorePath string
)

var _ = BeforeEach(func() {
	var err error

	cmd := exec.Command("lsmod")
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	session.Wait(5 * time.Second)
	if !strings.Contains(string(session.Out.Contents()), "ifb") {
		Skip("Docker for Mac does not contain IFB kernel module")
	}

	ifbName = fmt.Sprintf("i-some-ifb-%d", GinkgoParallelNode())
	dummyName = fmt.Sprintf("ilololol-%d", GinkgoParallelNode())
	notSilkCreatedIFBName = fmt.Sprintf("other-ifb-%d", GinkgoParallelNode())

	netlinkAdapter = &adapter.NetlinkAdapter{}

	// /var/vcap/data/container-metadata
	datastorePath, err = ioutil.TempDir(os.TempDir(), fmt.Sprintf("container-metadata-%d", GinkgoParallelNode()))
	Expect(err).NotTo(HaveOccurred())

	// /var/vcap/data/host-local
	delegateDataDirPath, err = ioutil.TempDir(os.TempDir(), fmt.Sprintf("host-local-%d", GinkgoParallelNode()))
	Expect(err).NotTo(HaveOccurred())

	// /var/vcap/data/silk/store.json
	delegateDatastorePath, err = ioutil.TempDir(os.TempDir(), fmt.Sprintf("silk-%d", GinkgoParallelNode()))
	Expect(err).NotTo(HaveOccurred())

	config = &lib.WrapperConfig{
		Datastore: filepath.Join(datastorePath, "store.json"),
		Delegate: map[string]interface{}{
			"dataDir":   delegateDataDirPath,
			"datastore": filepath.Join(delegateDatastorePath, "store.json"),
		},
		IPTablesLockFile:              "does_not_matter",
		InstanceAddress:               "does_not_matter",
		IngressTag:                    "does_not_matter",
		VTEPName:                      "does_not_matter",
		IPTablesDeniedLogsPerSec:      2,
		IPTablesAcceptedUDPLogsPerSec: 2,
	}
	// write config, pass it as flag to when we call teardown
	configFilePath = writeConfigFile(*config)

	mustSucceed("ip", "link", "add", ifbName, "type", "ifb")
	mustSucceed("ip", "link", "add", notSilkCreatedIFBName, "type", "ifb")
	mustSucceed("ip", "link", "add", dummyName, "type", "dummy")
})

var _ = AfterEach(func() {
	exec.Command("ip", "link", "del", ifbName).Run()
	mustSucceed("ip", "link", "del", notSilkCreatedIFBName)
	mustSucceed("ip", "link", "del", dummyName)

	Expect(os.RemoveAll(configFilePath)).To(Succeed())
})

var _ = Describe("Teardown", func() {
	It("destroys only leftover IFB devices and removes the unneeded directories", func() {
		By("running teardown")
		session := runTeardown()
		Expect(session).To(gexec.Exit(0))
		Expect(session.Out.Contents()).To(ContainSubstring("cni-teardown.starting"))

		By("verifying that the ifb is no longer present")
		_, err := netlinkAdapter.LinkByName(ifbName)
		Expect(err).To(MatchError("Link not found"))

		By("verifying that the other devices are not cleaned up")
		_, err = netlinkAdapter.LinkByName(dummyName)
		Expect(err).NotTo(HaveOccurred())

		_, err = netlinkAdapter.LinkByName(notSilkCreatedIFBName)
		Expect(err).NotTo(HaveOccurred())

		By("verifying that the relevant directories no longer exist")
		Expect(fileExists(datastorePath)).To(BeFalse())
		Expect(fileExists(delegateDataDirPath)).To(BeFalse())
		Expect(fileExists(delegateDatastorePath)).To(BeFalse())

		Expect(session.Out.Contents()).To(ContainSubstring("cni-teardown.complete"))
	})

	Context("when the config file does not exist", func() {
		BeforeEach(func() {
			configFilePath = "some/bad/path"
		})

		It("logs the errors and exits 1", func() {
			session := runTeardown()
			Expect(session).To(gexec.Exit(1))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.starting"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.load-config-file"))
			Expect(string(session.Out.Contents())).NotTo(ContainSubstring("cni-teardown.complete"))

		})
	})

	Context("when the config file exists but cannot be read", func() {
		BeforeEach(func() {
			err := ioutil.WriteFile(configFilePath, []byte("some-bad-data"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
		})
		It("logs the errors but still cleans up devices", func() {
			session := runTeardown()
			Expect(session).To(gexec.Exit(1))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.starting"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.read-config-file"))
			Expect(string(session.Out.Contents())).NotTo(ContainSubstring("cni-teardown.complete"))
		})
	})

	Context("when we fail to clean up the directories", func() {
		BeforeEach(func() {
			silkJsonPath := filepath.Join(delegateDatastorePath, "store.json")
			metadataJsonPath := filepath.Join(datastorePath, "store.json")
			hostLocalJsonPath := filepath.Join(delegateDataDirPath, "store.json")

			makeImmutableFile(silkJsonPath)
			makeImmutableFile(metadataJsonPath)
			makeImmutableFile(hostLocalJsonPath)
		})

		It("logs the errors but still cleans up devices", func() {
			By("running teardown")
			session := runTeardown()
			Expect(session).To(gexec.Exit(0))

			By("verifying that the ifb is no longer present")
			_, err := netlinkAdapter.LinkByName(ifbName)
			Expect(err).To(MatchError("Link not found"))

			By("checking the logs")
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.starting"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.failed-to-remove-datastore-path"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.failed-to-remove-delegate-datastore-path"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.failed-to-remove-delegate-data-dir-path"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.complete"))
		})
	})

	Context("when unable to delete an ifb device", func() {
		BeforeEach(func() {
			err := os.Chmod(configFilePath, 0777)
			Expect(err).NotTo(HaveOccurred())

			createUserCmd := exec.Command("useradd", "test-user")
			session, err := gexec.Start(createUserCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		AfterEach(func() {
			delUserCmd := exec.Command("deluser", "test-user")
			session, err := gexec.Start(delUserCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		It("logs the errors", func() {
			By("running teardown")
			session := runTeardownNonRoot("test-user")
			Expect(session).To(gexec.Exit(0))

			By("checking the logs")
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.starting"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("cni-teardown.failed-to-remove-ifb"))
		})
	})
})

func makeImmutableFile(fileName string) {
	_, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0400)
	Expect(err).NotTo(HaveOccurred())

	cmd := exec.Command("chattr", "+i", fileName)
	userAddSession, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(userAddSession, 5*time.Second).Should(gexec.Exit(0))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
func writeConfigFile(config lib.WrapperConfig) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}

func mustSucceed(binary string, args ...string) string {
	cmd := exec.Command(binary, args...)
	sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, "10s").Should(gexec.Exit(0))
	return string(sess.Out.Contents())
}

func runTeardown() *gexec.Session {
	startCmd := exec.Command(paths.TeardownBin, "--config", configFilePath)
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	return session
}

func runTeardownNonRoot(user string) *gexec.Session {
	startCmd := exec.Command("su", user, "-c", fmt.Sprintf("%s --config %s", paths.TeardownBin, configFilePath))
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	return session
}
