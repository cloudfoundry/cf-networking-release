package acceptance_test

import (
	"net/http"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Delete Acceptance", func() {
	var (
		prefix     string
		orgName    string
		srcAppName string
		dstAppName string
		hostName   string
	)

	BeforeEach(func() {
		prefix = "delete-sd-apps-"

		orgName = prefix + "org"
		spaceName := prefix + "space"
		srcAppName = prefix + "src-app-proxy"
		dstAppName = prefix + "dst-app-proxy"

		createAndTargetOrgAndSpace(orgName, spaceName)

		By("pushing the app and checking it resolves")
		pushApp(srcAppName, 1)
		pushApp(dstAppName, 1)

		Expect(cf.Cf("map-route", dstAppName, internalDomain, "--hostname", dstAppName).Wait(10 * time.Second)).To(gexec.Exit(0))
		hostName = "http://" + srcAppName + "." + config.AppsDomain + "/dig/" + dstAppName + "." + internalDomain
		proxyIPs := digForNumberOfIPs(hostName, 1)

		Expect(proxyIPs).To(ContainElement(getInternalIP(dstAppName, 0)))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Cf)).To(gexec.Exit(0))
	})

	Describe("when performing a dns lookup for a domain configured to point to the bosh adapter", func() {
		It("does not resolve the app hostname (returns a 500) when the app is deleted", func() {
			By("deleting the app")
			deleteApp(dstAppName)

			By("checking that the app is no longer resolved")
			Eventually(func() int {
				resp, err := http.Get(hostName)

				Expect(err).NotTo(HaveOccurred())
				return resp.StatusCode
			}, 5*time.Second).Should(Equal(http.StatusInternalServerError))
		})
	})
})

func deleteApp(appName string) {
	Expect(cf.Cf(
		"delete", appName, "-f",
	).Wait(Timeout_Cf)).To(gexec.Exit(0))
}
