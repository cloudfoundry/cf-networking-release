package acceptance_test

import (
	"os"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
)

var _ = Describe("trace logging for the plugin", func() {
	var (
		prefix    string
		orgName   string
		spaceName string
	)

	createAndTarget := func(orgName, spaceName string) {
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
	}

	BeforeEach(func() {
		prefix = testConfig.Prefix
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("when tracing is disabled", func() {
		BeforeEach(func() {
			orgName = prefix + "cli-plugin-trace-disabled-org"
			spaceName = prefix + "space"
			createAndTarget(orgName, spaceName)
		})

		It("does not log the HTTP request or response", func() {
			listAccess := cf.Cf("list-access")
			Expect(listAccess.Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(string(listAccess.Out.Contents())).NotTo(ContainSubstring("GET /networking/v0/external/policies"))
		})
	})

	Describe("when tracing is enabled", func() {
		BeforeEach(func() {
			orgName = prefix + "cli-plugin-trace-enabled-org"
			spaceName = prefix + "space"
			createAndTarget(orgName, spaceName)

			Expect(os.Setenv("CF_TRACE", "true")).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Unsetenv("CF_TRACE")).To(Succeed())
		})

		It("logs the HTTP request and responses to the policy server", func() {
			listAccess := cf.Cf("list-access")
			Expect(listAccess.Wait(Timeout_Push)).To(gexec.Exit(0))

			By("printing trace info", func() {
				Expect(string(listAccess.Out.Contents())).To(ContainSubstring("GET /networking/v0/external/policies"))
			})

			By("not printing private data", func() {
				Expect(string(listAccess.Out.Contents())).ToNot(ContainSubstring("bearer"))
			})
		})
	})
})
