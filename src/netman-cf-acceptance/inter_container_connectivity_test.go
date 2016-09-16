package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Push = 5 * time.Minute
const Timeout_Short = 10 * time.Second

var ports []int

func getSubnet(ip string) string {
	return strings.Split(ip, ".")[2]
}

func isSameCell(sourceIP, destIP string) bool {
	return getSubnet(sourceIP) == getSubnet(destIP)
}

var _ = Describe("connectivity between containers on the overlay network", func() {
	Describe("networking policy", func() {
		var (
			appProxy     string
			appRegistry  string
			appsTest     []string
			orgName      string
			spaceName    string
			appInstances int
			applications int
		)

		BeforeEach(func() {
			appInstances = testConfig.AppInstances
			applications = testConfig.Applications

			appProxy = fmt.Sprintf("proxy-%d", rand.Int31())
			appRegistry = fmt.Sprintf("registry-%d", rand.Int31())
			for i := 0; i < applications; i++ {
				appsTest = append(appsTest, fmt.Sprintf("tick-%d-%d", i, rand.Int31()))
			}

			ports = []int{8080}
			for i := 0; i < testConfig.Policies; i++ {
				ports = append(ports, 7000+i)
			}

			Auth(testConfig.TestUser, testConfig.TestUserPassword)

			orgName = "test-org"
			Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

			spaceName = "test-space"
			Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
		})

		AfterEach(func() {
			appReport(appProxy, Timeout_Short)
			appReport(appRegistry, Timeout_Short)
			appsReport(appsTest, Timeout_Short)

			// clean up everything
			Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		})

		It("allows the user to configure policies", func(done Done) {
			By("pushing the registry app and proxy app")
			var setupWG sync.WaitGroup
			setupWG.Add(2)
			go func() {
				defer GinkgoRecover()
				pushRegistryApp(appRegistry)
				setupWG.Done()
			}()

			go func() {
				defer GinkgoRecover()
				pushApp(appProxy)
				setupWG.Done()
			}()
			setupWG.Wait()

			By("pushing the tick apps")
			newManifest := modifyTickManifest(appRegistry)
			pushAppsOfType(appsTest, "tick", newManifest)

			By("scaling the tick apps")
			scaleApps(appsTest, appInstances)

			By("checking that all test app instances have registered themselves")
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest)*appInstances)

			appIPs := getAppIPs(appRegistry)

			By("checking that the connection fails")
			assertConnectionFails(appProxy, appIPs, ports)

			By("creating policies")
			for _, app := range appsTest {
				for _, port := range ports {
					session := cf.Cf("access-allow", appProxy, app, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(2 * Timeout_Short)
					Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))
				}
			}

			By(fmt.Sprintf("checking that %s can reach %s", appProxy, appsTest))
			assertConnectionSucceeds(appProxy, appIPs, ports)

			dumpStats(appProxy, config.AppsDomain)

			By("deleting policies")
			for _, app := range appsTest {
				for _, port := range ports {
					session := cf.Cf("access-deny", appProxy, app, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(2 * Timeout_Short)
					Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))
				}
			}

			By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsTest))
			assertConnectionFails(appProxy, appIPs, ports)

			By("checking that reflex no longer reports deleted instances")
			scaleApps(appsTest, 1 /* instances */)
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest))

			close(done)
		}, 10*60 /* <-- overall spec timeout in seconds */)
	})
})

func dumpStats(host, domain string) {
	resp, err := http.Get(fmt.Sprintf("http://%s.%s/stats", host, domain))
	Expect(err).NotTo(HaveOccurred())
	respBytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	fmt.Printf("STATS: %s\n", string(respBytes))
	netStatsFile := os.Getenv("NETWORK_STATS_FILE")
	if netStatsFile != "" {
		Expect(ioutil.WriteFile(netStatsFile, respBytes, 0600)).To(Succeed())
	}
}

func checkRegistry(registry string, timeout, pollingInterval time.Duration, totalInstances int) {
	registeredApps := func() (int, error) {
		resp, err := http.Get(fmt.Sprintf("http://%s.%s/api/v1/instances", registry, config.AppsDomain))
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		var instancesResponse struct {
			Instances []struct {
				ServiceName string `json:"service_name"`
			} `json:"instances"`
		}
		err = json.Unmarshal(respBytes, &instancesResponse)
		if err != nil {
			return 0, err
		}
		return len(instancesResponse.Instances), nil
	}

	Eventually(registeredApps, timeout, pollingInterval).Should(Equal(totalInstances))
}

func getAppIPs(registry string) []string {
	resp, err := http.Get(fmt.Sprintf("http://%s.%s/api/v1/instances", registry, config.AppsDomain))
	Expect(err).NotTo(HaveOccurred())
	respBytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	var instancesResponse struct {
		Instances []struct {
			Endpoint struct {
				Value string `json:"value"`
			} `json:"endpoint"`
		} `json:"instances"`
	}
	Expect(json.Unmarshal(respBytes, &instancesResponse)).To(Succeed())
	ips := []string{}
	for _, instance := range instancesResponse.Instances {
		ip, _, err := net.SplitHostPort(instance.Endpoint.Value)
		Expect(err).NotTo(HaveOccurred())
		ips = append(ips, ip)
	}
	return ips
}

func assertConnectionSucceeds(sourceApp string, destApps []string, ports []int) {
	workPoolRun(destApps, func(appIP string) {
		for _, port := range ports {
			assertSingleConnection(appIP, port, sourceApp, true)
		}
	})
}

func assertConnectionFails(sourceApp string, destApps []string, ports []int) {
	workPoolRun(destApps, func(appIP string) {
		for _, port := range ports {
			assertSingleConnection(appIP, port, sourceApp, false)
		}
	})
}

func assertSingleConnection(destIP string, port int, sourceAppName string, shouldSucceed bool) {
	proxyTest := func() (string, error) {
		resp, err := http.Get(fmt.Sprintf("http://%s.%s/proxy/%s:%d", sourceAppName, config.AppsDomain, destIP, port))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(respBytes), nil
	}
	if shouldSucceed {
		By(fmt.Sprintf("eventually proxy should reach %s at port %d", destIP, port))
		Eventually(proxyTest, 10*time.Second, 500*time.Millisecond).ShouldNot(ContainSubstring("failed"))
	} else {
		By(fmt.Sprintf("eventually proxy should NOT reach %s at port %d", destIP, port))
		Eventually(proxyTest, 10*time.Second, 500*time.Millisecond).Should(ContainSubstring("request failed"))
	}
}
