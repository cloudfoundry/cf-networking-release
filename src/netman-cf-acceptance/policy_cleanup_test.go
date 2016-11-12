package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"fmt"
	"math/rand"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("policy cleanup", func() {
	var (
		appA      string
		orgName   string
		spaceName string
		cfCli     *cf_cli_adapter.Adapter
	)

	BeforeEach(func() {
		appA = fmt.Sprintf("appA-%d", rand.Int31())

		cfCli = &cf_cli_adapter.Adapter{
			CfCliPath: "cf",
		}
		AuthAsAdmin()

		orgName = "cleanup-org"
		Expect(cfCli.CreateOrg(orgName)).To(Succeed())
		Expect(cfCli.TargetOrg(orgName)).To(Succeed())

		spaceName = "cleanup-space"
		Expect(cfCli.CreateSpace(spaceName)).To(Succeed())
		Expect(cfCli.TargetSpace(spaceName)).To(Succeed())

		pushProxy(appA)
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("policies/cleanup endpoint", func() {
		It("cleans up stale policies for deleted apps", func() {
			By("creating a policy")
			Expect(cfCli.AllowAccess(appA, appA, 1234, "tcp")).To(Succeed())

			appAGuid, err := cfCli.AppGuid(appA)
			Expect(err).NotTo(HaveOccurred())

			By("checking that policy exists")
			allPolicies, err := cfCli.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(appAGuid))

			By("cleaning up stale policies")
			stalePolicies, err := cfCli.Curl("POST", "/networking/v0/external/policies/cleanup", "")
			Expect(string(stalePolicies)).ShouldNot(ContainSubstring(appAGuid))

			By("checking that policy was not deleted")
			allPolicies, err = cfCli.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(appAGuid))

			By("deleting app so policy becomes stale")
			Expect(cfCli.Delete(appA)).To(Succeed())

			By("cleaning up stale policies")
			stalePolicies, err = cfCli.Curl("POST", "/networking/v0/external/policies/cleanup", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stalePolicies)).Should(ContainSubstring(appAGuid))

			By("checking that stale policy was deleted")
			allPolicies, err = cfCli.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).ShouldNot(ContainSubstring(appAGuid))
		})
	})
})
