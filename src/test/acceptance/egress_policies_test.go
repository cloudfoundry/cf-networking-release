package acceptance_test

import (
	"cf-pusher/cf_cli_adapter"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"

	"regexp"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("external connectivity", func() {
	var (
		appA                               string
		orgName                            string
		spaceName                          string
		appRoute                           string
		wideOpenTCPplusICMPDestinationGuid string
		wideOpenAllDestinationGuid         string
		egressPolicyGuid                   string
		cli                                *cf_cli_adapter.Adapter
		testWideOpenTCPplusICMPDestination = `{
			"destinations": [
				{
					"name": "egress-test-wide-open-tcp-with-some-icmp-%s",
					"description": "This is to test external connectivity with a wide open tcp policy and some icmp.",
					"rules": [
						{
							"protocol": "tcp",
							"ports": [ { "start": 1, "end": 65535 } ],
							"ips": [ { "start": "0.0.0.0", "end": "255.255.255.255" } ]
						},
						{
							"protocol": "icmp",
							"ips": [ { "start": "1.1.1.1", "end": "1.1.1.1" } ]
						},
						{
							"protocol": "icmp",
							"ips": [ { "start": "8.8.8.8", "end": "8.8.8.8" } ]
						}
					]
				}
			]
		}`
		testWideOpenAllDestination = `{
			"destinations": [
				{
					"name": "egress-test-all-protocols-%s",
					"description": "This is to test external connectivity to all protocols with a single rule.",
					"rules": [
						{
							"protocol": "all",
							"ports": [ { "start": 1, "end": 65535 } ],
							"ips": [ { "start": "0.0.0.0", "end": "255.255.255.255" } ]
						}
					]
				}
			]
		}`
		testStagingEgressPolicies = `{
			"egress_policies": [ {
					"source": { "id": %q, "type": %q },
					"destination": { "id": %q },
					"app_lifecycle": "staging"
				} ]
		}`
		testRunningEgressPolicies = `{
			"egress_policies": [ {
					"source": { "id": %q, "type": %q },
					"destination": { "id": %q },
					"app_lifecycle": "running"
				} ]
		}`
		testAllEgressPolicies = `{
			"egress_policies": [ {
					"source": { "id": %q, "type": %q },
					"destination": { "id": %q },
					"app_lifecycle": "all"
				} ]
		}`
	)

	BeforeEach(func() {
		if testConfig.Internetless || testConfig.SkipExperimentalDynamicEgressTest {
			Skip("skipping egress policy tests")
		}

		cli = &cf_cli_adapter.Adapter{CfCliPath: "cf"}
		appA = fmt.Sprintf("appA-%d", rand.Int31())

		orgName = testConfig.Prefix + "egress-policy-org"
		spaceName = testConfig.Prefix + "space"
		setupOrgAndSpace(orgName, spaceName)

		By("unbinding all running ASGs")
		for _, sg := range testConfig.DefaultSecurityGroups {
			Expect(cf.Cf("unbind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		By("unbinding all staging ASGs")
		for _, sg := range testConfig.DefaultSecurityGroups {
			Expect(cf.Cf("unbind-staging-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		By("creating all destinations")
		wideOpenTCPplusICMPDestinationGuid = createDestination(cli, fmt.Sprintf(testWideOpenTCPplusICMPDestination, fmt.Sprintf("%d", rand.Int31())))
		wideOpenAllDestinationGuid = createDestination(cli, fmt.Sprintf(testWideOpenAllDestination, fmt.Sprintf("%d", rand.Int31())))
	})

	AfterEach(func() {
		By("deleting destinations")
		deleteDestination(cli, wideOpenTCPplusICMPDestinationGuid)
		deleteDestination(cli, wideOpenAllDestinationGuid)

		By("adding back all the original staging ASGs")
		for _, sg := range testConfig.DefaultSecurityGroups {
			Expect(cf.Cf("bind-staging-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		By("adding back all the original running ASGs")
		for _, sg := range testConfig.DefaultSecurityGroups {
			Expect(cf.Cf("bind-running-security-group", sg).Wait(Timeout_Short)).To(gexec.Exit(0))
		}

		By("deleting the test org")
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
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

	canDigUDP := func() error {
		return checkRequest(appRoute+"digudp/example.com", 200, `93.184.216.34`)
	}

	cannotDigUDP := func() error {
		return checkRequest(appRoute+"digudp/example.com", 500, `Failed to dig`)
	}

	canProxy := func() error {
		return checkRequest(appRoute+"proxy/example.com", 200, `Example Domain`)
	}
	cannotProxy := func() error {
		return checkRequest(appRoute+"proxy/example.com", 500, "connection refused|i/o timeout")
	}

	canPing := func(ipAddress string) error {
		return checkRequest(appRoute+"ping/"+ipAddress, 200, `Ping succeeded to destination: `+ipAddress)
	}

	Context("when an app lifecycle 'all' egress policy is created", func() {
		BeforeEach(func() {
			By("creating all egress policy")
			spaceGuid, err := cli.SpaceGuid(spaceName)
			Expect(err).NotTo(HaveOccurred())
			egressPolicyGuid = createEgressPolicy(cli, fmt.Sprintf(testAllEgressPolicies, spaceGuid, "space", wideOpenTCPplusICMPDestinationGuid))
		})

		AfterEach(func() {
			By("deleting all egress policy")
			deleteEgressPolicy(cli, egressPolicyGuid)
		})

		It("the app can reach the internet when egress policy is present", func(done Done) {
			By("pushing the test app")
			pushProxy(appA)
			appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)

			By("checking that the app can use dns and http to reach the internet")
			Eventually(canProxy, "10s", "1s").Should(Succeed())
			Consistently(canProxy, "2s", "0.5s").Should(Succeed())

			Consistently(func() error { return canPing("8.8.8.8") }, "2s", "0.5s").Should(Succeed())
			Consistently(func() error { return canPing("1.1.1.1") }, "2s", "0.5s").Should(Succeed())

			By("checking that the app cannot use UDP")
			Consistently(cannotDigUDP, "2s", "0.5s").Should(Succeed())

			close(done)
		}, 180 /* <-- overall spec timeout in seconds */)
	})

	Context("when a protocol 'all' egress policy is created", func() {
		BeforeEach(func() {
			By("creating all egress policy")
			spaceGuid, err := cli.SpaceGuid(spaceName)
			Expect(err).NotTo(HaveOccurred())
			egressPolicyGuid = createEgressPolicy(cli, fmt.Sprintf(testAllEgressPolicies, spaceGuid, "space", wideOpenAllDestinationGuid))
		})

		AfterEach(func() {
			By("deleting all egress policy")
			deleteEgressPolicy(cli, egressPolicyGuid)
		})

		It("the app can reach the internet when egress policy is present", func(done Done) {
			By("pushing the test app")
			pushProxy(appA)
			appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)

			By("checking that the app can use dns and http to reach the internet")
			Eventually(canProxy, "10s", "1s").Should(Succeed())
			Consistently(canProxy, "2s", "0.5s").Should(Succeed())

			Consistently(func() error { return canPing("8.8.8.8") }, "2s", "0.5s").Should(Succeed())
			Consistently(func() error { return canPing("1.1.1.1") }, "2s", "0.5s").Should(Succeed())

			Eventually(canDigUDP, "10s", "1s").Should(Succeed())
			Consistently(canDigUDP, "2s", "0.5s").Should(Succeed())

			close(done)
		}, 180 /* <-- overall spec timeout in seconds */)
	})

	Context("when a staging egress policy is created", func() {
		BeforeEach(func() {
			By("creating staging egress policy")
			spaceGuid, err := cli.SpaceGuid(spaceName)
			Expect(err).NotTo(HaveOccurred())
			egressPolicyGuid = createEgressPolicy(cli, fmt.Sprintf(testStagingEgressPolicies, spaceGuid, "space", wideOpenTCPplusICMPDestinationGuid))
		})

		AfterEach(func() {
			By("deleting staging egress policy")
			deleteEgressPolicy(cli, egressPolicyGuid)
		})

		Context("when the egress policy is for the app", func() {
			var (
				egressPolicyGuid string
			)

			AfterEach(func() {
				By("deleting egress policy")
				deleteEgressPolicy(cli, egressPolicyGuid)

				By("checking that the app cannot reach the internet using http and dns")
				Eventually(cannotProxy, "10s", "1s").Should(Succeed())
				Consistently(cannotProxy, "2s", "0.5s").Should(Succeed())
			})

			It("the app can reach the internet when egress policy is present", func(done Done) {
				By("pushing the test app")
				pushProxy(appA)
				appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)

				By("checking that the app cannot reach the internet using http and dns")
				Eventually(cannotProxy, "10s", "1s").Should(Succeed())
				Consistently(cannotProxy, "2s", "0.5s").Should(Succeed())

				By("creating running egress policy")
				appAGuid, err := cli.AppGuid(appA)
				Expect(err).NotTo(HaveOccurred())
				egressPolicyGuid = createEgressPolicy(cli, fmt.Sprintf(testRunningEgressPolicies, appAGuid, "app", wideOpenTCPplusICMPDestinationGuid))

				By("checking that the app can use dns and http to reach the internet")
				Eventually(canProxy, "10s", "1s").Should(Succeed())
				Consistently(canProxy, "2s", "0.5s").Should(Succeed())

				close(done)
			}, 180 /* <-- overall spec timeout in seconds */)
		})

		Context("when the egress policy is for the space", func() {
			var (
				egressPolicyGuid string
			)

			AfterEach(func() {
				By("deleting egress policy")
				deleteEgressPolicy(cli, egressPolicyGuid)

				By("checking that the app cannot reach the internet using http and dns")
				Eventually(cannotProxy, "10s", "1s").Should(Succeed())
				Consistently(cannotProxy, "2s", "0.5s").Should(Succeed())
			})

			It("the app in the space can reach the internet when egress policy is present", func(done Done) {
				By("pushing the test app")
				pushProxy(appA)
				appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)

				By("checking that the space cannot reach the internet using http and dns")
				Eventually(cannotProxy, "10s", "1s").Should(Succeed())
				Consistently(cannotProxy, "2s", "0.5s").Should(Succeed())

				By("creating running egress policy")
				spaceGuid, err := cli.SpaceGuid(spaceName)
				Expect(err).NotTo(HaveOccurred())
				egressPolicyGuid = createEgressPolicy(cli, fmt.Sprintf(testRunningEgressPolicies, spaceGuid, "space", wideOpenTCPplusICMPDestinationGuid))

				By("checking that the app can use dns and http to reach the internet")
				Eventually(canProxy, "10s", "1s").Should(Succeed())
				Consistently(canProxy, "2s", "0.5s").Should(Succeed())

				close(done)
			}, 180 /* <-- overall spec timeout in seconds */)
		})

		Context("when a policy is already applied to the space", func() {
			var (
				egressPolicyGuid string
			)

			BeforeEach(func() {
				By("creating an egress policy")
				spaceGuid, err := cli.SpaceGuid(spaceName)
				Expect(err).NotTo(HaveOccurred())
				egressPolicyGuid = createEgressPolicy(cli, fmt.Sprintf(testRunningEgressPolicies, spaceGuid, "space", wideOpenTCPplusICMPDestinationGuid))
			})

			AfterEach(func() {
				By("deleting egress policy")
				deleteEgressPolicy(cli, egressPolicyGuid)
			})

			It("the app in the space can reach the internet immediately after a push", func(done Done) {
				By("pushing the test app")
				pushProxy(appA)
				appRoute = fmt.Sprintf("http://%s.%s/", appA, config.AppsDomain)

				Expect(canProxy()).To(Succeed())

				close(done)
			}, 180 /* <-- overall spec timeout in seconds */)
		})
	})
})

func deleteEgressPolicy(cli *cf_cli_adapter.Adapter, guid string) {
	var egressPolicyDeleteStruct struct {
		Error string `json:"error"`
	}

	response, err := cli.Curl("DELETE", fmt.Sprintf("/networking/v1/external/egress_policies/%s", guid), "")
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(response, &egressPolicyDeleteStruct)
	Expect(err).NotTo(HaveOccurred())
	Expect(egressPolicyDeleteStruct.Error).To(BeEmpty())
}

func createEgressPolicy(cli *cf_cli_adapter.Adapter, payload string) string {
	payloadFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())

	var egressPolicyStruct struct {
		EgressPolicies []struct {
			ID string `json:"id"`
		} `json:"egress_policies"`
		Error string `json:"error"`
	}

	_, err = payloadFile.Write([]byte(payload))
	Expect(err).NotTo(HaveOccurred())

	err = payloadFile.Close()
	Expect(err).NotTo(HaveOccurred())

	response, err := cli.Curl("POST", "/networking/v1/external/egress_policies", payloadFile.Name())
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(response, &egressPolicyStruct)
	Expect(err).NotTo(HaveOccurred())
	Expect(egressPolicyStruct.Error).To(BeEmpty())

	err = os.Remove(payloadFile.Name())
	Expect(err).NotTo(HaveOccurred())
	return egressPolicyStruct.EgressPolicies[0].ID
}

func createDestination(cli *cf_cli_adapter.Adapter, payload string) string {
	payloadFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())

	var destStruct struct {
		Destinations []struct {
			ID string `json:"id"`
		} `json:"destinations"`
		Error string `json:"error"`
	}

	_, err = payloadFile.Write([]byte(payload))
	Expect(err).NotTo(HaveOccurred())

	err = payloadFile.Close()
	Expect(err).NotTo(HaveOccurred())

	response, err := cli.Curl("POST", "/networking/v1/external/destinations", payloadFile.Name())
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(response, &destStruct)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("cannot unmarshal json: %s", response))
	Expect(destStruct.Error).To(BeEmpty(), destStruct.Error)

	err = os.Remove(payloadFile.Name())
	Expect(err).NotTo(HaveOccurred())

	return destStruct.Destinations[0].ID
}

func deleteDestination(cli *cf_cli_adapter.Adapter, guid string) {
	var destDeleteStruct struct {
		Error string `json:"error"`
	}

	response, err := cli.Curl("DELETE", fmt.Sprintf("/networking/v1/external/destinations/%s", guid), "")
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(response, &destDeleteStruct)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("cannot unmarshal json: %s", response))
	Expect(destDeleteStruct.Error).To(BeEmpty())
}
