package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	"code.cloudfoundry.org/lib/testsupport"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("external connectivity", func() {
	var (
		appA      string
		orgName   string
		spaceName string
		appRoute  string
	)

	BeforeEach(func() {
		if testConfig.Internetless {
			Skip("skipping external connectivity tests")
		}

		appA = fmt.Sprintf("appA-%d", rand.Int31())

		orgName = testConfig.Prefix + "external-connectivity-org"
		spaceName = testConfig.Prefix + "space"
		setupOrgAndSpace(orgName, spaceName)

		By("unbinding all running ASGs")
		for _, sg := range testConfig.DefaultSecurityGroups {
			Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		By("creating test-generated ASGs")
		for asgName, asgValue := range testASGs {
			createASG(asgName, asgValue)
		}

		By("pushing the test app")
		pushProxy(appA)
		appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)
	})

	AfterEach(func() {
		By("adding back all the original running ASGs")
		for _, sg := range testConfig.DefaultSecurityGroups {
			Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		By("deleting the test org")
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))

		By("removing test-generated ASGs")
		for asgName, _ := range testASGs {
			removeASG(asgName)
		}
	})

	checkRequest := func(route string, expectedStatusCode int, expectedResponseRegex string) error {
		regex := regexp.MustCompile(expectedResponseRegex)
		resp, err := http.Get(route)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		respBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		respBody := string(respBytes)

		if resp.StatusCode != expectedStatusCode {
			return fmt.Errorf("test http get to %s: expected response code %d but got %d.  response body:\n%s", route, expectedStatusCode, resp.StatusCode, respBody)
		}
		if !regex.MatchString(respBody) {
			return fmt.Errorf("test http get to %s: expected response to contain %q but instead saw:\n%s", route, expectedResponseRegex, respBody)
		}
		return nil
	}

	isReachable := func() error {
		return checkRequest(appRoute, 200, `{"ListenAddresses":\[`)
	}
	canProxy := func() error {
		return checkRequest(appRoute+"proxy/docs.cloudfoundry.org", 200, `https://docs\.cloudfoundry\.org`)
	}
	cannotProxy := func() error {
		return checkRequest(appRoute+"proxy/docs.cloudfoundry.org", 500, "connection refused|i/o timeout")
	}
	canPing := func() error {
		return checkRequest(appRoute+"ping/8.8.8.8", 200, "Ping succeeded")
	}
	cannotPing := func() error {
		return checkRequest(appRoute+"ping/8.8.8.8", 500, `Ping failed to destination: 8\.8\.8\.8`)
	}

	Describe("basic (legacy) network behavior for an app", func() {
		It("makes the app reachable from the router, and the app can reach the internet only if allowed", func(done Done) {
			By("checking that the app is reachable via the router")
			Eventually(isReachable, "10s", "1s").Should(Succeed())
			Consistently(isReachable, "2s", "0.5s").Should(Succeed())

			By("checking that the app cannot reach the internet using http and dns")
			Eventually(cannotProxy, "10s", "1s").Should(Succeed())
			Consistently(cannotProxy, "2s", "0.5s").Should(Succeed())

			By("checking that the app cannot ping the internet (first time)")
			Consistently(cannotPing, "2s", "0.5s").Should(Succeed())

			By("creating and binding a tcp and udp security group")
			Expect(cfCLI.BindSecurityGroup("tcp-asg", orgName, spaceName)).To(Succeed())
			Expect(cfCLI.BindSecurityGroup("udp-asg", orgName, spaceName)).To(Succeed())

			if !testConfig.DynamicASGsEnabled {
				By("if dynamic asgs are not enabled, restarting the app is required")
				Expect(cf.Cf("restart", appA).Wait(Timeout_Push)).To(gexec.Exit(0))
			}

			By("checking that the app can use dns and http to reach the internet")
			Eventually(canProxy, "180s", "1s").Should(Succeed())
			Consistently(canProxy, "2s", "0.5s").Should(Succeed())

			By("checking that the app cannot ping the internet (second time)")
			Consistently(cannotPing, "2s", "1s").Should(Succeed())

			By("removing the tcp security groups")
			Expect(cfCLI.UnbindSecurityGroup("tcp-asg", orgName, spaceName)).To(Succeed())

			if !testConfig.DynamicASGsEnabled {
				By("restarting the app")
				Expect(cf.Cf("restart", appA).Wait(Timeout_Push)).To(gexec.Exit(0))
			} else {
				time.Sleep(10 * time.Second)
			}
			By("checking that the app cannot use http to reach the internet")
			Consistently(cannotProxy, "180s", "0.5s").Should(Succeed())

			close(done)
		}, 180 /* <-- overall spec timeout in seconds */)

		It("allows outbound ICMP only if allowed", func(done Done) {
			if testConfig.SkipICMPTests {
				Skip("Test config has 'skip_icmp_test: true', skipping ICMP connectivity tests")
			}

			By("checking that the app cannot ping the internet")
			Consistently(cannotPing, "2s", "0.5s").Should(Succeed())

			By("creating and binding an icmp security group")
			Expect(cfCLI.BindSecurityGroup("icmp-asg", orgName, spaceName)).To(Succeed())

			if !testConfig.DynamicASGsEnabled {
				By("restarting the app")
				Expect(cf.Cf("restart", appA).Wait(Timeout_Push)).To(gexec.Exit(0))
			}

			By("checking that the app can ping the internet")
			Eventually(canPing, "180s", "1s").Should(Succeed())
			Consistently(canPing, "2s", "0.5s").Should(Succeed())

			close(done)
		}, 180 /* <-- overall spec timeout in seconds */)
	})
})

func createASG(name string, asgDefinition string) {
	asgFile, err := testsupport.CreateTempFile(asgDefinition)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfCLI.CreateSecurityGroup(name, asgFile)).To(Succeed())
	Expect(os.Remove(asgFile)).To(Succeed())
}

func removeASG(name string) {
	Expect(cfCLI.DeleteSecurityGroup(name)).To(Succeed())
}

func setupOrgAndSpace(orgName, spaceName string) {
	Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
	Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

	Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
	Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
}

var testASGs = map[string]string{
	"tcp-asg": `
	[
		{
			"destination": "0.0.0.0-255.255.255.255",
			"protocol": "tcp",
			"ports": "80,443"
		}
	]
	`,
	"udp-asg": `
	[
		{
			"destination": "0.0.0.0-255.255.255.255",
			"protocol": "udp",
			"ports": "53"
		}
	]
	`,
	"icmp-asg": `
	[
		{
			"destination": "0.0.0.0-255.255.255.255",
			"protocol": "icmp",
			"type": 8,
			"code": 0
		}
	]
	`,
}
