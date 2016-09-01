package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
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
			appsReflex   []string
			orgName      string
			spaceName    string
			appInstances int
			applications int
		)

		BeforeEach(func() {
			appInstances = testConfig.AppInstances
			applications = testConfig.Applications

			appProxy = fmt.Sprintf("proxy-%d", rand.Int31())
			for i := 0; i < applications; i++ {
				appsReflex = append(appsReflex, fmt.Sprintf("reflex-%d-%d", i, rand.Int31()))
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

			pushApp(appProxy)

			newManifest := modifyReflexManifest()
			pushAppsOfType(appsReflex, "reflex", newManifest)
			for _, app := range appsReflex {
				By("creating a new policy to allow the reflex app to talk to itself")
				session := cf.Cf("access-allow", app, app, "--protocol", "tcp", "--port", fmt.Sprintf("%d", ports[0])).Wait(2 * Timeout_Short)
				Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))
				scaleApp(app, appInstances)
			}
		})

		AfterEach(func() {
			appReport(appProxy, Timeout_Short)
			appsReport(appsReflex, Timeout_Short)

			// clean up everything
			Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		})

		It("allows the user to configure connections", func(done Done) {
			By("checking that the reflex app has discovered all its instances")
			checkPeers(appsReflex, 60*time.Second, 500*time.Millisecond, appInstances)

			By("checking that the connection fails")
			assertConnectionFails(appProxy, appsReflex, ports)

			By("creating a new policy")
			for _, app := range appsReflex {
				for _, port := range ports {
					session := cf.Cf("access-allow", appProxy, app, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(2 * Timeout_Short)
					Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))
				}
			}

			By(fmt.Sprintf("checking that %s can reach %s", appProxy, appsReflex))
			assertConnectionSucceeds(appProxy, appsReflex, ports)

			dumpStats(appProxy, config.AppsDomain)

			By("deleting the policy")
			for _, app := range appsReflex {
				for _, port := range ports {
					session := cf.Cf("access-deny", appProxy, app, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(2 * Timeout_Short)
					Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))
				}
			}

			By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsReflex))
			assertConnectionFails(appProxy, appsReflex, ports)

			By("checking that reflex no longer reports deleted instances")
			scaleApps(appsReflex, 1 /* instances */)
			checkPeers(appsReflex, 60*time.Second, 500*time.Millisecond, appInstances)

			close(done)
		}, 900 /* <-- overall spec timeout in seconds */)
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

func checkPeers(apps []string, timeout, pollingInterval time.Duration, instances int) {
	for _, app := range apps {
		getPeers := func() ([]string, error) {
			resp, err := http.Get(fmt.Sprintf("http://%s.%s/peers", app, config.AppsDomain))
			if err != nil {
				return nil, err
			}

			respBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			var peersResponse struct {
				IPs []string
			}
			err = json.Unmarshal(respBytes, &peersResponse)
			if err != nil {
				return nil, err
			}
			return peersResponse.IPs, nil
		}

		Eventually(getPeers, timeout, pollingInterval).Should(HaveLen(instances))
	}
}

func assertConnectionSucceeds(sourceApp string, destApps []string, ports []int) {
	for _, app := range destApps {
		for _, port := range ports {
			assertAllConnectionStatus(sourceApp, app, port, true)
		}
	}
}

func assertConnectionFails(sourceApp string, destApps []string, ports []int) {
	for _, app := range destApps {
		for _, port := range ports {
			assertAllConnectionStatus(sourceApp, app, port, false)
		}
	}
}

func assertAllConnectionStatus(sourceApp, destApp string, port int, shouldSucceed bool) {
	resp, err := http.Get(fmt.Sprintf("http://%s.%s/peers", destApp, config.AppsDomain))
	Expect(err).NotTo(HaveOccurred())
	respBytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	var addressListJson struct {
		IPs []string
	}
	Expect(json.Unmarshal(respBytes, &addressListJson)).To(Succeed())

	for _, destIP := range addressListJson.IPs {
		assertSingleConnection(sourceApp, destIP, port, shouldSucceed)
	}
}

func assertSingleConnection(sourceAppName string, destIP string, port int, shouldSucceed bool) {
	proxyTest := func() (string, error) {
		resp, err := http.Get(fmt.Sprintf("http://%s.%s/proxy/%s:%d/peers", sourceAppName, config.AppsDomain, destIP, port))
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
		Eventually(proxyTest, 10*time.Second).ShouldNot(ContainSubstring("failed"))
	} else {
		Eventually(proxyTest, 10*time.Second).Should(ContainSubstring("request failed"))
	}
}
