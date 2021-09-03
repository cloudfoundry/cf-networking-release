package acceptance_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	spamAPI "example-apps/spammer/api"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	orgName     = "outbound-conn-limit-test-org"
	spaceName   = "outbound-conn-limit-test-space"
	proxyName   = "proxy"
	spammerName = "spammer"

	setEnvTimeoutInSec = 10
	burst              = 60
)

var _ = Describe("Outbound connection limit", func() {
	BeforeEach(func() {
		if testConfig.RunExperimentalOutboundConnLimitTest {
			Skip("Skipping outbound connection limit test")
		}

		AuthAsAdmin()

		Expect(cfCLI.CreateOrg(orgName)).To(Succeed())
		Expect(cfCLI.TargetOrg(orgName)).To(Succeed())

		Expect(cfCLI.CreateSpace(spaceName, orgName)).To(Succeed())
		Expect(cfCLI.TargetSpace(spaceName)).To(Succeed())
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("when an app opens multiple connections to one host", func() {
		It("the connections get rate limited", func() {
			By("pushing proxy")
			pushProxy(proxyName)

			By("pushing spammer")
			pushSpammer()

			By("verifying the burst is available on start")
			spamResp := spam()
			Expect(spamResp.SuccessCount).Should(BeEquivalentTo(burst))

			By("verifying the burst is exhausted")
			spamResp = spam()
			Expect(spamResp.SuccessCount).Should(BeNumerically("<", burst))
		})
	})
})

func pushSpammer() {
	session := cf.Cf("push", spammerName, "-p", appDir(spammerName), "-f", defaultManifest(spammerName), "--no-start")
	Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))

	proxyBaseURL := getAppBaseURL(proxyName)
	session = cf.Cf("set-env", spammerName, spamAPI.ProxyBaseURLField, proxyBaseURL)
	Expect(session.Wait(setEnvTimeoutInSec)).To(gexec.Exit(0))

	session = cf.Cf("start", spammerName)
	Expect(session.Wait(Timeout_Push)).To(gexec.Exit(0))
}

func spam() *spamAPI.SpamResp {
	spammerBaseURL := spamEndpoint(burst)
	resp, err := http.Get(spammerBaseURL)

	Expect(err).Should(BeNil())
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	target := &spamAPI.SpamResp{}

	Expect(decoder.Decode(target)).Should(BeNil())

	return target
}

func spamEndpoint(callCount int) string {
	spammerBaseURL := getAppBaseURL(spammerName)
	return fmt.Sprintf("%s%s%d", spammerBaseURL, spamAPI.SpamPath, callCount)
}

func getAppBaseURL(appName string) string {
	return fmt.Sprintf("https://%s.%s", appName, testConfig.AppsDomain)
}
