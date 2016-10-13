package integration_test

import (
	"cf-pusher/config"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

const DEFAULT_TIMEOUT = "10s"

var cfPusher string

var _ = BeforeSuite(func() {
	// only run on node 1
	fmt.Fprintf(GinkgoWriter, "building binary...")
	var err error
	cfPusher, err = gexec.Build("cf-pusher/cmd/cf-pusher", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func WriteConfigFile(config config.Config) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}
