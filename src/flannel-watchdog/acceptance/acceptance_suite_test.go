package acceptance_test

import (
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const DEFAULT_TIMEOUT = "5s"

var watchdogBinaryPath string

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
}

var _ = BeforeSuite(func() {
	// only run on node 1
	fmt.Fprintf(GinkgoWriter, "building binary...")
	var err error
	watchdogBinaryPath, err = gexec.Build("flannel-watchdog/cmd/flannel-watchdog", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
