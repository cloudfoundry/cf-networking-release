package acceptance_test

import (
	"fmt"
	"math/rand"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("policy cleanup", func() {
	var (
		appA      string
		orgName   string
		spaceName string
	)

	BeforeEach(func() {
		appA = fmt.Sprintf("appA-%d", rand.Int31())

		AuthAsAdmin()

		orgName = "cleanup-org"
		Expect(cfCLI.CreateOrg(orgName)).To(Succeed())
		Expect(cfCLI.TargetOrg(orgName)).To(Succeed())

		spaceName = "cleanup-space"
		Expect(cfCLI.CreateSpace(spaceName, orgName)).To(Succeed())
		Expect(cfCLI.TargetSpace(spaceName)).To(Succeed())

		pushProxy(appA)
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		_, err := cfCLI.CleanupStaleNetworkPolicies()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("policies/cleanup endpoint", func() {
		It("cleans up stale policies for deleted apps", func() {
			By("creating a policy")
			Expect(cfCLI.AddNetworkPolicy(appA, appA, 1234, "tcp")).To(Succeed())

			appAGuid, err := cfCLI.AppGuid(appA)
			Expect(err).NotTo(HaveOccurred())

			By("checking that policy exists")
			allPolicies, err := cfCLI.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(appAGuid))

			By("cleaning up stale policies")
			stalePolicies, err := cfCLI.CleanupStaleNetworkPolicies()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stalePolicies)).ShouldNot(ContainSubstring(appAGuid))

			By("checking that policy was not deleted")
			allPolicies, err = cfCLI.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(appAGuid))

			By("deleting app so policy becomes stale")
			Expect(cfCLI.Delete(appA)).To(Succeed())

			By("cleaning up stale policies")
			stalePolicies, err = cfCLI.CleanupStaleNetworkPolicies()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stalePolicies)).Should(ContainSubstring(appAGuid))

			By("checking that stale policy was deleted")
			allPolicies, err = cfCLI.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).ShouldNot(ContainSubstring(appAGuid))
		})
	})
})
