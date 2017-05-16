package timeouts_test

import (
	"fmt"
	"math/rand"
	"policy-server/integration/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	ginkgoConfig "github.com/onsi/ginkgo/config"

	"testing"
)

var policyServerPath string

var MockCCServer = helpers.MockCCServer
var MockUAAServer = helpers.MockUAAServer

func TestTimeouts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Timeouts Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	fmt.Fprintf(GinkgoWriter, "building binary...")
	policyServerPath, err := gexec.Build("policy-server/cmd/policy-server", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	return []byte(policyServerPath)
}, func(data []byte) {
	policyServerPath = string(data)
	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
