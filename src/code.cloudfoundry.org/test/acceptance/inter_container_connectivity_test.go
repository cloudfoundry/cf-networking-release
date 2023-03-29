package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	helpers "github.com/cloudfoundry/cf-test-helpers/v2/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Short = 10 * time.Second
const PolicyWaitTime = 7 * time.Second

var ports []int

var _ = Describe("connectivity between containers on the overlay network", func() {
	Describe("networking policy", func() {
		var (
			appsProxy         []string
			appRegistry       string
			appsTest          []string
			appInstances      int
			applications      int
			proxyApplications int
			proxyInstances    int
			prefix            string
			orgName           string
		)

		BeforeEach(func() {
			prefix = testConfig.Prefix

			orgName = prefix + "org" // cf-pusher expects this name
			Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

			spaceName := prefix + "space" // cf-pusher expects this name
			Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

			appInstances = testConfig.AppInstances
			applications = testConfig.Applications
			proxyApplications = testConfig.ProxyApplications
			proxyInstances = testConfig.ProxyInstances

			for i := 0; i < proxyApplications; i++ {
				appsProxy = append(appsProxy, fmt.Sprintf(prefix+"proxy-%d", i))
			}
			appRegistry = prefix + "registry"
			for i := 0; i < applications; i++ {
				appsTest = append(appsTest, fmt.Sprintf(prefix+"tick-%d", i))
			}

			ports = []int{8080}
			for i := 0; i < testConfig.ExtraListenPorts; i++ {
				ports = append(ports, 7000+i)
			}
		})

		AfterEach(func() {
			Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
			_, err := cfCLI.CleanupStaleNetworkPolicies()
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows policies to whitelist traffic between applications", func(done Done) {
			cmd := exec.Command("go", "run", "../../cf-pusher/cmd/cf-pusher/main.go", "--config", helpers.ConfigPath())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Println("\n----- cf-pusher start ------")
			err := cmd.Run()
			fmt.Println("\n----- cf-pusher done -------")
			Expect(err).NotTo(HaveOccurred())

			By("checking that all test app instances have registered themselves")
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest)*appInstances)

			appIPs := getAppIPs(appRegistry)

			By("checking that the connection fails")
			for _, appProxy := range appsProxy {
				By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsTest))
				assertConnectionsFail(appProxy, appIPs, ports, proxyInstances)
			}

			By("creating policies")
			for _, appProxy := range appsProxy {
				createAllPolicies(appProxy, appsTest, ports)
			}

			// we should wait for minimum (pollInterval * 2)
			By(fmt.Sprintf("waiting %s for policies to be created on cells", PolicyWaitTime))
			time.Sleep(PolicyWaitTime)

			By("checking that the connection succeeds")
			for _, appProxy := range appsProxy {
				By(fmt.Sprintf("checking that %s can reach %s", appProxy, appsTest))
				assertConnectionsSucceed(appProxy, appIPs, ports, proxyInstances)
			}

			By("deleting policies")
			for _, appProxy := range appsProxy {
				deleteAllPolicies(appProxy, appsTest, ports)
			}

			By(fmt.Sprintf("waiting %s for policies to be deleted on cells", PolicyWaitTime))
			time.Sleep(PolicyWaitTime)

			By("checking that the connection fails, again")
			for _, appProxy := range appsProxy {
				By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsTest))
				assertConnectionsFail(appProxy, appIPs, ports, proxyInstances)
			}

			By("checking that the registry updates when apps are scaled")
			scaleApps(appsTest, 1 /* instances */)
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest))

			scaleApps(appsTest, appInstances /* instances */)
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest)*appInstances)

			close(done)
		}, 30*60 /* <-- overall spec timeout in seconds */)
	})
})

func createAllPolicies(sourceApp string, dstList []string, dstPorts []int) {
	for _, destApp := range dstList {
		for _, port := range dstPorts {
			err := cfCLI.AddNetworkPolicy(sourceApp, destApp, port, "tcp")
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func deleteAllPolicies(sourceApp string, dstList []string, dstPorts []int) {
	for _, destApp := range dstList {
		for _, port := range dstPorts {
			err := cfCLI.RemoveNetworkPolicy(sourceApp, destApp, port, "tcp")
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

type RegistryInstancesResponse struct {
	Instances []struct {
		ServiceName string `json:"service_name"`
		Endpoint    struct {
			Value string `json:"value"`
		} `json:"endpoint"`
	} `json:"instances"`
}

func getInstancesFromA8(registry string) (*RegistryInstancesResponse, error) {
	resp, err := httpGetBytes(fmt.Sprintf("http://%s.%s/api/v1/instances", registry, config.AppsDomain))
	if err != nil {
		return nil, err
	}

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var instancesResponse RegistryInstancesResponse
	err = json.Unmarshal(resp.Body, &instancesResponse)
	if err != nil {
		return nil, err
	}
	return &instancesResponse, nil
}

func checkRegistry(registry string, timeout, pollingInterval time.Duration, totalInstances int) {
	registeredApps := func() (int, error) {
		instancesResponse, err := getInstancesFromA8(registry)
		if err != nil {
			return 0, err
		}
		return len(instancesResponse.Instances), nil
	}

	Eventually(registeredApps, timeout, pollingInterval).Should(Equal(totalInstances))
}

func getAppIPs(registry string) []string {
	instancesResponse, err := getInstancesFromA8(registry)
	Expect(err).NotTo(HaveOccurred())

	ips := []string{}
	for _, instance := range instancesResponse.Instances {
		ip, _, err := net.SplitHostPort(instance.Endpoint.Value)
		Expect(err).NotTo(HaveOccurred())
		ips = append(ips, ip)
	}
	return ips
}

func assertConnectionsSucceed(sourceApp string, destApps []string, ports []int, nProxies int) {
	assertConnections(sourceApp, destApps, ports, nProxies, true)
}

func assertConnectionsFail(sourceApp string, destApps []string, ports []int, nProxies int) {
	assertConnections(sourceApp, destApps, ports, nProxies, false)
}

func assertConnections(sourceApp string, destApps []string, ports []int, nProxies int, shouldSucceed bool) {
	for _, appIP := range destApps {
		for _, port := range ports {
			assertSingleConnection(appIP, port, sourceApp, shouldSucceed)
		}
	}
}

func assertSingleConnection(destIP string, port int, sourceAppName string, shouldSucceed bool) {
	if shouldSucceed {
		By(fmt.Sprintf("eventually proxy should reach %s at port %d", destIP, port))
		assertResponseContains(destIP, port, sourceAppName, "application_name")
	} else {
		By(fmt.Sprintf("eventually proxy should NOT reach %s at port %d", destIP, port))
		assertResponseContains(destIP, port, sourceAppName, "request failed")
	}
}

func assertResponseContains(destIP string, port int, sourceAppName string, desiredResponse string) {
	proxyTest := func() (string, error) {
		resp, err := httpGetBytes(fmt.Sprintf("http://%s.%s/proxy/%s:%d", sourceAppName, config.AppsDomain, destIP, port))
		if err != nil {
			return "", err
		}
		return string(resp.Body), nil
	}
	Eventually(proxyTest, 10*time.Second, 500*time.Millisecond).Should(ContainSubstring(desiredResponse))
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout:   4 * time.Second,
			KeepAlive: 0,
		}).Dial,
	},
	Timeout: 20 * time.Second,
}

type httpResp struct {
	StatusCode int
	Body       []byte
}

func httpGetBytes(url string) (httpResp, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return httpResp{}, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return httpResp{}, err
	}

	return httpResp{resp.StatusCode, respBytes}, nil
}
