package integration_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var (
	paths testPaths
)

type testPaths struct {
	PathToAdapter       string
	PathToFakeCNIPlugin string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error
	paths.PathToAdapter, err = gexec.Build("code.cloudfoundry.org/garden-external-networker", "-race")
	Expect(err).NotTo(HaveOccurred())

	paths.PathToFakeCNIPlugin, err = gexec.Build("code.cloudfoundry.org/garden-external-networker/integration/fake-cni-plugin", "-race")
	Expect(err).NotTo(HaveOccurred())

	data, err := json.Marshal(paths)
	Expect(err).NotTo(HaveOccurred())

	return data
}, func(data []byte) {
	Expect(json.Unmarshal(data, &paths)).To(Succeed())

	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelProcess()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
