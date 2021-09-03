package acceptance_test

import (
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
			stalePolicies, err := cfCLI.Curl("POST", "/networking/v0/external/policies/cleanup", "")
			Expect(string(stalePolicies)).ShouldNot(ContainSubstring(appAGuid))

			By("checking that policy was not deleted")
			allPolicies, err = cfCLI.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(appAGuid))

			By("deleting app so policy becomes stale")
			Expect(cfCLI.Delete(appA)).To(Succeed())

			By("cleaning up stale policies")
			stalePolicies, err = cfCLI.Curl("POST", "/networking/v0/external/policies/cleanup", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stalePolicies)).Should(ContainSubstring(appAGuid))

			By("checking that stale policy was deleted")
			allPolicies, err = cfCLI.Curl("GET", "/networking/v0/external/policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).ShouldNot(ContainSubstring(appAGuid))
		})

		It("cleans up stale policies for deleted spaces", func() {
			By("creating a destination")
			testDestination := `{
				"destinations": [
					{
						"name": %q,
						"description": "Testing description",
						"rules": [
							{
								"protocol": "tcp",
								"ports": "80-80",
								"ips": "0.0.0.0-255.255.255.255"
							}
						]
					}
				]
			}`
			destinationGuid := createDestination(fmt.Sprintf(testDestination, fmt.Sprintf("egress-policies-%d", rand.Int31())))

			By("creating an egress policy for a space")
			testEgressPolicies := `{
				"egress_policies": [ {
						"source": { "id": %q, "type": %q },
						"destination": { "id": %q }
					} ]
			}`
			spaceGuid, err := cfCLI.SpaceGuid(spaceName)
			Expect(err).NotTo(HaveOccurred())
			createEgressPolicy(fmt.Sprintf(testEgressPolicies, spaceGuid, "space", destinationGuid))

			By("checking that policy exists")
			allPolicies, err := cfCLI.Curl("GET", "/networking/v1/external/egress_policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(spaceGuid))

			By("cleaning up stale policies")
			stalePolicies, err := cfCLI.Curl("POST", "/networking/v0/external/policies/cleanup", "")
			Expect(string(stalePolicies)).ShouldNot(ContainSubstring(spaceGuid))

			By("checking that policy was not deleted")
			allPolicies, err = cfCLI.Curl("GET", "/networking/v1/external/egress_policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).Should(ContainSubstring(spaceGuid))

			By("deleting space so policy becomes stale")
			Expect(cf.Cf("delete-space", spaceName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))

			By("cleaning up stale policies")
			stalePolicies, err = cfCLI.Curl("POST", "/networking/v0/external/policies/cleanup", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stalePolicies)).Should(ContainSubstring(spaceGuid))

			By("checking that stale policy was deleted")
			allPolicies, err = cfCLI.Curl("GET", "/networking/v1/external/egress_policies", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(allPolicies)).ShouldNot(ContainSubstring(spaceGuid))
		})
	})
})
