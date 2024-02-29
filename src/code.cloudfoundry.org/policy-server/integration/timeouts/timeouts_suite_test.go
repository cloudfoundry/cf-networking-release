package timeouts_test

import (
	"fmt"
	"testing"

	"code.cloudfoundry.org/policy-server/integration/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
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
	policyServerPath, err := gexec.Build("code.cloudfoundry.org/policy-server/cmd/policy-server", "-race", "-buildvcs=false")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	return []byte(policyServerPath)
}, func(data []byte) {
	policyServerPath = string(data)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
