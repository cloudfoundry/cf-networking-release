package acceptance_test

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Cf = 2 * time.Minute
const domain = "apps.internal"

var _ = Describe("Push Acceptance", func() {
	var (
		prefix  string
		orgName string
		appName string
	)

	BeforeEach(func() {
		prefix = "push-sd-apps-"

		orgName = prefix + "org"
		spaceName := prefix + "space"
		appName = prefix + "proxy"

		createAndTargetOrgAndSpace(orgName, spaceName)
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Cf)).To(gexec.Exit(0))
	})

	Describe("when performing a dns lookup for a domain configured to point to the bosh adapter", func() {
		It("returns the result from the adapter", func() {
			pushApp(appName, 1)
			Expect(cf.Cf("map-route", appName, domain, "--hostname", appName).Wait(10 * time.Second)).To(gexec.Exit(0))

			hostName := "http://" + appName + "." + config.AppsDomain + "/dig/" + appName + "." + domain
			proxyIPs := digForNumberOfIPs(hostName, 1)

			Expect(proxyIPs).To(ContainElement(getInternalIP(appName, 0)))
		})
	})
})
