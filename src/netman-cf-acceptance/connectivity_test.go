package acceptance_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Push = 5 * time.Minute
const Timeout_Short = 10 * time.Second

func getSubnet(ip string) string {
	return strings.Split(ip, ".")[2]
}

var _ = Describe("connectivity tests", func() {
	var (
		appA                string
		appB                string
		appAIP              string
		appAGuid            string
		appBGuid            string
		remoteContainerIP   string
		sameCellContainerIP string
		orgName             string
		spaceName           string
	)

	BeforeEach(func() {
		appA = "appA"
		appB = "appB"

		orgName = "test-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceName = "test-space"
		Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

		pushApp(appA)

		instances := 4
		pushApp(appB)
		scaleApp(appB, instances)

		// wait for ssh to become available on new instances
		time.Sleep(5000 * time.Millisecond)

		appAGuid = getAppGuid(appA)
		appBGuid = getAppGuid(appB)

		appAIP = getInstanceIP(appA, 0)

		inDifferentCells := false
		for i := 0; i < instances; i++ {
			instanceIP := getInstanceIP(appB, i)

			if getSubnet(appAIP) != getSubnet(instanceIP) {
				inDifferentCells = true
				remoteContainerIP = instanceIP
			} else {
				sameCellContainerIP = instanceIP
			}
		}
		Expect(inDifferentCells).To(BeTrue())
	})

	AfterEach(func() {
		// clean up everything
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("networking policy", func() {
		It("makes an app reachable via its external route", func() {
			Eventually(func() string {
				return helpers.CurlAppWithTimeout(appA, "/", 6*Timeout_Short)
			}, 6*Timeout_Short).Should(ContainSubstring(appAIP))
		})

		Context("when the user is network admin", func() {
			var policyJSON string

			It("allows the user to configure connections", func() {
				By("by denying inter-cell communication by default")
				Consistently(func() string {
					return curlFromApp(appA, 0, fmt.Sprintf("%s:%d/", remoteContainerIP, 8080), false)
				}, Timeout_Short).ShouldNot(ContainSubstring(remoteContainerIP))

				By("by denying intra-cell communication by default")
				Consistently(func() string {
					return curlFromApp(appA, 0, fmt.Sprintf("%s:%d/", sameCellContainerIP, 8080), false)
				}, Timeout_Short).ShouldNot(ContainSubstring(sameCellContainerIP))

				Auth(testConfig.TestUser, testConfig.TestUserPassword)
				By("creating a new policy")
				policyJSON = fmt.Sprintf(`{"policies":[{"source":{"id":"%s"},"destination":{"id":"%s","protocol":"tcp","port":8080}}]}`,
					appAGuid,
					appBGuid,
				)
				curlSession := cf.Cf("curl", "-X", "POST", "/networking/v0/external/policies", "-d", "'"+policyJSON+"'").Wait(Timeout_Push)
				Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
				postPolicyOut := string(curlSession.Out.Contents())
				Expect(postPolicyOut).To(MatchJSON(`{}`))

				AuthAsAdmin()
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

				By("checking that an app on the same cell becomes reachable at its **internal** route")
				Eventually(func() string {
					return curlFromApp(appA, 0, fmt.Sprintf("%s:%d/", sameCellContainerIP, 8080), true)
				}, 6*Timeout_Short).Should(ContainSubstring(sameCellContainerIP))

				Auth(testConfig.TestUser, testConfig.TestUserPassword)
				By("deleting the policy")
				curlSession = cf.Cf("curl", "-X", "DELETE", "/networking/v0/external/policies", "-d", "'"+policyJSON+"'").Wait(Timeout_Push)
				Expect(curlSession.Wait(Timeout_Push)).To(gexec.Exit(0))
				deletePolicyOut := string(curlSession.Out.Contents())
				Expect(deletePolicyOut).To(MatchJSON(`{}`))

				AuthAsAdmin()
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

				By("checking that the app is no longer reachable")
				time.Sleep(5 * time.Second)
				Consistently(func() string {
					return curlFromApp(appA, 0, fmt.Sprintf("%s:%d/", sameCellContainerIP, 8080), false)
				}, Timeout_Short).ShouldNot(ContainSubstring(sameCellContainerIP))
			})
		})
	})
})
