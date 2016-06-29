package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"netman-agent/config"
	"os"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const DEFAULT_TIMEOUT = "5s"

var netmanAgentPath string

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
}

var _ = BeforeSuite(func() {
	fmt.Fprintf(GinkgoWriter, "building binary...")
	var err error
	netmanAgentPath, err = gexec.Build("netman-agent/cmd/netman-agent", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func WriteConfigFile(netmanAgentConfig config.Config) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(netmanAgentConfig)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}
