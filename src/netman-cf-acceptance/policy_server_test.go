package acceptance_test

import (
	"fmt"
	"math/rand"
	"os/exec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("policy server tests", func() {
	It("makes the policy server available at an external route", func() {
		cmd := exec.Command("curl", "-v", fmt.Sprintf("http://%s/networking", config.ApiEndpoint))

		sess, err := gexec.Start(cmd, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))

		curlOutput := sess.Out.Contents()
		Expect(curlOutput).To(ContainSubstring("Network policy server, up for"))
	})

	Context("When the user has network.admin scope", func() {
		BeforeEach(func() {
			Auth(testConfig.TestUser, testConfig.TestUserPassword)
		})
		AfterEach(func() {
			AuthAsAdmin()
		})

		It("allows access to whoami", func() {
			curlSession := cf.Cf("curl", "/networking/v0/external/whoami").Wait(Timeout_Push)

			Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
			whoamiOut := string(curlSession.Out.Contents())
			Expect(whoamiOut).To(MatchJSON(fmt.Sprintf(`{"user_name":%q}`, testConfig.TestUser)))
		})

		It("allows users to post, get and delete policies", func() {
			appGuid := rand.Int()

			policyJSON := fmt.Sprintf(`{"policies":[{"source":{"id":"%d"},"destination":{"id":"some-other-app-guid","protocol":"tcp","port":3333}}]}`, appGuid)

			By("creating a new policy")
			curlSession := cf.Cf("curl", "-X", "POST", "/networking/v0/external/policies", "-d", "'"+policyJSON+"'").Wait(Timeout_Push)
			Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
			postPolicyOut := string(curlSession.Out.Contents())
			Expect(postPolicyOut).To(MatchJSON(`{}`))

			By("getting the new policy")
			curlSession = cf.Cf("curl", "-X", "GET", fmt.Sprintf("/networking/v0/external/policies?id=%d", appGuid)).Wait(Timeout_Push)
			Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
			getPolicyOut := string(curlSession.Out.Contents())
			Expect(getPolicyOut).To(MatchJSON(policyJSON))

			By("deleting the policy")
			curlSession = cf.Cf("curl", "-X", "DELETE", "/networking/v0/external/policies", "-d", "'"+policyJSON+"'").Wait(Timeout_Push)
			Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
			deletePolicyOut := string(curlSession.Out.Contents())
			Expect(deletePolicyOut).To(MatchJSON(`{}`))

			By("checking that the policy no longer exists")
			curlSession = cf.Cf("curl", "-X", "GET", fmt.Sprintf("/networking/v0/external/policies?id=%d", appGuid)).Wait(Timeout_Push)
			Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
			getPolicyOut = string(curlSession.Out.Contents())
			Expect(getPolicyOut).To(MatchJSON(`{ "policies": [] }`))
		})
	})

})
