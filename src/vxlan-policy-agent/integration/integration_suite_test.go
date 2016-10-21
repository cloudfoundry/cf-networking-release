package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/testsupport"
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
	certDir string
	paths   testPaths
)

type testPaths struct {
	ServerCACertFile     string
	ClientCACertFile     string
	ServerCertFile       string
	ServerKeyFile        string
	ClientCertFile       string
	ClientKeyFile        string
	VxlanPolicyAgentPath string
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error
	certDir, err = ioutil.TempDir("", "netman-certs")
	Expect(err).NotTo(HaveOccurred())

	certstrapBin := fmt.Sprintf("/%s/certstrap", certDir)
	cmd := exec.Command("go", "build", "-o", certstrapBin, "github.com/square/certstrap")
	Expect(cmd.Run()).NotTo(HaveOccurred())

	certWriter := &testsupport.CertWriter{
		BinPath:  certstrapBin,
		CertPath: certDir,
	}

	paths.ServerCACertFile, err = certWriter.WriteCA("server-ca")
	Expect(err).NotTo(HaveOccurred())
	paths.ServerCertFile, paths.ServerKeyFile, err = certWriter.WriteAndSignForServer("server", "server-ca")
	Expect(err).NotTo(HaveOccurred())

	paths.ClientCACertFile, err = certWriter.WriteCA("client-ca")
	Expect(err).NotTo(HaveOccurred())
	paths.ClientCertFile, paths.ClientKeyFile, err = certWriter.WriteAndSignForClient("client", "client-ca")
	Expect(err).NotTo(HaveOccurred())

	fmt.Fprintf(GinkgoWriter, "building binary...")
	paths.VxlanPolicyAgentPath, err = gexec.Build("vxlan-policy-agent/cmd/vxlan-policy-agent", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())

	return data
}, func(data []byte) {
	Expect(json.Unmarshal(data, &paths)).To(Succeed())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
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
