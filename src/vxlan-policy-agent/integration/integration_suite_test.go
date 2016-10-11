package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	. "lib/testsupport"
	"math/rand"
	"os"
	"os/exec"
	"vxlan-policy-agent/config"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var DEFAULT_TIMEOUT = "5s"

var (
	certstrapBin         string
	certDir              string
	serverCACertPath     string
	clientCACertPath     string
	serverCertPath       string
	serverKeyPath        string
	clientCertPath       string
	clientKeyPath        string
	vxlanPolicyAgentPath string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))

	var err error
	certDir, err = ioutil.TempDir("", "netman-certs")
	Expect(err).NotTo(HaveOccurred())

	certstrapBin = fmt.Sprintf("/%s/certstrap", certDir)
	cmd := exec.Command("go", "build", "-o", certstrapBin, "github.com/square/certstrap")
	Expect(cmd.Run()).NotTo(HaveOccurred())

	serverCACertPath, err = WriteCACert(certstrapBin, certDir, "server-ca")
	Expect(err).NotTo(HaveOccurred())

	serverCertPath, serverKeyPath, err = WriteAndSignServerCert(certstrapBin, certDir, "server", "server-ca")
	Expect(err).NotTo(HaveOccurred())

	clientCACertPath, err = WriteCACert(certstrapBin, certDir, "client-ca")
	Expect(err).NotTo(HaveOccurred())

	clientCertPath, clientKeyPath, err = WriteAndSignServerCert(certstrapBin, certDir, "client", "client-ca")
	Expect(err).NotTo(HaveOccurred())

	fmt.Fprintf(GinkgoWriter, "building binary...")
	vxlanPolicyAgentPath, err = gexec.Build("vxlan-policy-agent/cmd/vxlan-policy-agent", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	os.Remove(certDir)
})

func WriteConfigFile(Config config.VxlanPolicyAgent) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(Config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}
