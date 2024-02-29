package acceptance_test

import (
	"fmt"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Custom iptables compatibility", func() {
	var (
		appName   string
		orgName   string
		spaceName string
	)

	BeforeEach(func() {
		if !testConfig.RunCustomIPTablesCompatibilityTest {
			Skip("skipping custom iptables compatibility tests")
		}

		appName = fmt.Sprintf("appA-%d", randomGenerator.Int31())

		orgName = testConfig.Prefix + "custom-iptables-org"
		spaceName = testConfig.Prefix + "space"
		setupOrgAndSpace(orgName, spaceName)

		By("pushing the test app")
		pushProxy(appName)
	})

	AfterEach(func() {
		By("deleting the test org")
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("when a custom iptables rule is added and a new app is pushed", func() {
		It("still applies the iptable rule to the new app", func(ctx SpecContext) {
			By("checking that the app can reach the process running on the host")
			session := cf.Cf("ssh", appName, "-c", "curl $CF_INSTANCE_IP:8898").Wait(10 * time.Second)
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Hello world!!"))
		}, SpecTimeout(3*time.Minute))
	})
})
