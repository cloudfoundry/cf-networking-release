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
		proxyApp   string
		proxyIP    string
		backendIP  string
		sameCellIP string
		orgName    string
	)

	BeforeEach(func() {
		proxyApp = "proxy-app"

		orgName = "test-org"
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		firstSpace := "space1"
		Expect(cf.Cf("create-space", firstSpace).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", firstSpace).Wait(Timeout_Push)).To(gexec.Exit(0))

		instances := 4
		pushApp(proxyApp)
		scaleApp(proxyApp, instances)

		// wait for ssh to become available on new instances
		time.Sleep(5000 * time.Millisecond)

		proxyIP = getInstanceIP(proxyApp, 0)

		inDifferentCells := false
		for i := 1; i < instances; i++ {
			instanceIP := getInstanceIP(proxyApp, i)

			if getSubnet(proxyIP) != getSubnet(instanceIP) {
				inDifferentCells = true
				backendIP = instanceIP
			} else {
				sameCellIP = instanceIP
			}
		}
		Expect(inDifferentCells).To(BeTrue())
	})

	AfterEach(func() {
		// clean up everything
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	It("makes everything reachable", func() {
		By("checking that the proxy is reachable via its external route")
		Eventually(func() string {
			return helpers.CurlAppWithTimeout(proxyApp, "/", 6*Timeout_Short)
		}, 6*Timeout_Short).Should(ContainSubstring(proxyIP))

		By("checking that the backend in a different cell is reachable via the proxy at its **internal** route")
		Eventually(func() string {
			return curlFromApp(proxyApp, 0, fmt.Sprintf("%s:%d/proxy", backendIP, 8080))
		}, 6*Timeout_Short).Should(ContainSubstring(backendIP))

		By("checking that the backend in the same cell is reachable via the proxy at its **internal** route")
		Eventually(func() string {
			return curlFromApp(proxyApp, 0, fmt.Sprintf("%s:%d/proxy", sameCellIP, 8080))
		}, 6*Timeout_Short).Should(ContainSubstring(sameCellIP))
	})
})
