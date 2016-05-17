package acceptance_test

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Push = 5 * time.Minute
const Timeout_Short = 10 * time.Second

var _ = Describe("connectivity tests", func() {
	var (
		proxyApp   string
		proxyIP    string
		backendApp string
		backendIP  string
		orgName    string
	)

	BeforeEach(func() {
		proxyApp = "proxy-app-1"
		backendApp = "backend-app"

		orgName = "test-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		firstSpace := "space1"
		Expect(cf.Cf("create-space", firstSpace).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", firstSpace).Wait(Timeout_Push)).To(gexec.Exit(0))

		pushApp(proxyApp)
		pushApp(backendApp)

		proxyIP = getInstanceIP(proxyApp, 0)
		backendIP = getInstanceIP(backendApp, 0)
	})

	AfterEach(func() {
		// clean up everything
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	It("makes everything reachable", func() {
		By("checking that the proxy is reachable via its external route")
		Eventually(func() string {
			return helpers.CurlAppWithTimeout(proxyApp, "/", 6*Timeout_Short)
		}, 6*Timeout_Short).Should(ContainSubstring("hello, this is proxy"))

		By("checking that the backend is reachable via its external route")
		Eventually(func() string {
			return helpers.CurlAppWithTimeout(backendApp, "/", 6*Timeout_Short)
		}, 6*Timeout_Short).Should(ContainSubstring("hello, this is proxy"))

		By("checking that the backend is reachable via the proxy at its **external** route")
		backendWithoutScheme := backendApp + "." + helpers.LoadConfig().AppsDomain
		Eventually(func() string {
			return helpers.CurlAppWithTimeout(proxyApp, "/proxy/"+backendWithoutScheme, 6*Timeout_Short)
		}, 6*Timeout_Short).Should(ContainSubstring("hello, this is proxy"))

		By("checking that the backend is reachable via the proxy at its **internal** route")
		Eventually(func() string {
			return helpers.CurlAppWithTimeout(proxyApp, "/proxy/"+backendIP+":8080", 6*Timeout_Short)
		}, 6*Timeout_Short).Should(ContainSubstring("hello, this is proxy"))
	})
})
