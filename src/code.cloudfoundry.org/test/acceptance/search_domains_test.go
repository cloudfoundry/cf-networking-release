package acceptance_test

import (
	"fmt"
	"math/rand"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// NOTE: This test assumes that garden-cni job has been configured to have `apps.internal` as a search domain
// see `manifest-generation/opsfiles/add-apps-internal-search-domain.yml`
var _ = Describe("search domains", func() {
	var (
		appName string
	)

	BeforeEach(func() {
		if testConfig.SkipSearchDomainTests {
			Skip("skipping search domains test")
		}

		appName = fmt.Sprintf("appName-%d", rand.Int31())

		By("pushing the test app")
		pushProxy(appName)
	})

	It("/etc/resolv.conf contains apps.internal", func() {
		session := cf.Cf("ssh", appName, "-c", "cat /etc/resolv.conf")
		Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(string(session.Out.Contents())).To(MatchRegexp("search.*apps\\.internal"))
	})
})
