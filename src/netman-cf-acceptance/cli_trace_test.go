package acceptance_test

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("enabling trace logging for the CF CLI plugin", func() {
	var (
		orgName   string
		spaceName string
	)

	BeforeEach(func() {
		prefix := testConfig.Prefix
		orgName = prefix + "org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceName = prefix + "space"
		Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	It("logs the HTTP request and responses to the policy server", func() {
		// run CF_TRACE=true cf list-access
		// assert that stdout includes HTTP GET to the right url

		listAccess := cf.Cf("-v", "list-access")
		Expect(listAccess.Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(string(listAccess.Out.Contents())).To(ContainSubstring("GET /networking/v0/external/policies"))
	})
})
