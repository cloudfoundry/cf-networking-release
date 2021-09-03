package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ASG/Dynamic Egress Interaction", func() {
	var (
		appA                 string
		orgName              string
		spaceName            string
		appRoute             string
		bothEgressPolicyGuid string
		onlyEgressPolicyGuid string
		bothDestinationGuid  string
		onlyDestinationGuid  string
		allASGs              = map[string]string{
			"both-asg": `[{
					"destination": "93.184.216.34",
					"protocol": "tcp",
					"ports": "80,443"
				}]
			`,
			"only-asg": `[{
					"destination": "208.80.154.224",
					"protocol": "tcp",
					"ports": "80,443"
				}]`,
		}
		overlappingDestination = `{
			"destinations": [ {
					"name": %q,
					"description": "Testing description",
					"rules": [
						{
							"protocol": "tcp",
							"ports": "80-80",
							"ips": "93.184.216.34-93.184.216.34"
						}
					]
				} ]
		}`
		nonOverlappingDestination = `{
			"destinations": [ {
					"name": %q,
					"description": "Testing description",
					"rules": [
						{
							"protocol": "tcp",
							"ports": "80-443",
							"ips": "198.35.26.96-198.35.26.96"
						}
					]
				} ]
		}`
		testEgressPolicies = `{
			"egress_policies": [ {
					"source": { "id": %q, "type": %q },
					"destination": { "id": %q }
				} ]
		}`
	)

	BeforeEach(func() {
		if testConfig.Internetless || testConfig.SkipExperimentalDynamicEgressTest {
			Skip("skipping asg/dynamic egress interaction tests")
		}

		appA = fmt.Sprintf("appA-%d", rand.Int31())

		orgName = testConfig.Prefix + "asg-de-interaction-org"
		spaceName = testConfig.Prefix + "space"
		setupOrgAndSpace(orgName, spaceName)

		By("unbinding all running ASGs")
		for _, sg := range testConfig.DefaultSecurityGroups {
			Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		By("creating test-generated ASGs")
		for asgName, asgValue := range allASGs {
			createASG(asgName, asgValue)
		}

		By("creating and binding a tcp and udp security group")
		Expect(cfCLI.BindSecurityGroup("both-asg", orgName, spaceName)).To(Succeed())
		Expect(cfCLI.BindSecurityGroup("only-asg", orgName, spaceName)).To(Succeed())

		By("creating dynamic egress policies to same destination as ASG")
		bothDestinationGuid = createDestination(fmt.Sprintf(overlappingDestination, fmt.Sprintf("asg-egress-overlap-%d", rand.Int31())))
		spaceGuid, err := cfCLI.SpaceGuid(spaceName)
		Expect(err).NotTo(HaveOccurred())
		bothEgressPolicyGuid = createEgressPolicy(fmt.Sprintf(testEgressPolicies, spaceGuid, "space", bothDestinationGuid))

		By("creating dynamic egress policies to different destination from ASG")
		onlyDestinationGuid = createDestination(fmt.Sprintf(nonOverlappingDestination, fmt.Sprintf("asg-egress-overlap-%d", rand.Int31())))
		onlyEgressPolicyGuid = createEgressPolicy(fmt.Sprintf(testEgressPolicies, spaceGuid, "space", onlyDestinationGuid))

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
		for asgName, _ := range allASGs {
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

	canProxy := func() error {
		return checkRequest(appRoute+"proxy/example.com", 200, `Example Domain`)
	}
	canProxyASGOnlySite := func() error {
		return checkRequest(appRoute+"proxy/208.80.154.224", 200, `wikimedia`)
	}
	canProxyDEOnlySite := func() error {
		return checkRequest(appRoute+"proxy/198.35.26.96", 200, `wikimedia`)
	}
	cannotProxyDEOnlySite := func() error {
		return checkRequest(appRoute+"proxy/198.35.26.96", 500, "connection refused|i/o timeout")
	}

	It("can reach all the websites allowed by both asgs and dynamic egress", func(done Done) {
		By("checking that the app can talk to the websites allowed by both")
		Eventually(canProxy, "10s", "1s").Should(Succeed())
		Eventually(canProxyASGOnlySite, "10s", "1s").Should(Succeed())
		Eventually(canProxyDEOnlySite, "10s", "1s").Should(Succeed())

		By("deleting all the dynamic egress policies")
		deleteEgressPolicy(bothEgressPolicyGuid)
		deleteEgressPolicy(onlyEgressPolicyGuid)
		deleteDestination(bothDestinationGuid)
		deleteDestination(onlyDestinationGuid)

		By("checking that the app can stil reach the websites allowed by the asgs")
		Eventually(canProxy, "10s", "1s").Should(Succeed())
		Eventually(canProxyASGOnlySite, "10s", "1s").Should(Succeed())

		By("checking that the app cannot reach the website previously allowed by dynamic egress policies")
		Eventually(cannotProxyDEOnlySite, "10s", "0.5s").Should(Succeed())

		close(done)
	}, 180 /* <-- overall spec timeout in seconds */)
})
