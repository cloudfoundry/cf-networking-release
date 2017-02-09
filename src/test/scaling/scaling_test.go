package scaling_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/models"
	"lib/policy_client"
	"lib/testsupport"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/rainmaker"
)

const Timeout_Check = 20 * time.Minute

// 2 * poll cycle time (5s)
const Policy_Update_Wait = 10 * time.Second

var ports []int

var _ = Describe("how the container network performs at scale", func() {
	Describe("networking policy", func() {
		var (
			appProxy       string
			appRegistry    string
			appsTest       []string
			appInstances   int
			applications   int
			proxyInstances int
			sampleSize     int
		)

		BeforeEach(func() {
			appInstances = pushConfig.AppInstances
			applications = pushConfig.Applications
			proxyInstances = pushConfig.ProxyInstances
			sampleSize = pushConfig.SampleSize
			appProxy = pushConfig.Prefix + "proxy"
			appRegistry = pushConfig.Prefix + "registry"
			for i := 0; i < applications; i++ {
				appsTest = append(appsTest, fmt.Sprintf(pushConfig.Prefix+"tick-%d", i))
			}

			ports = []int{8080}
			for i := 0; i < pushConfig.ExtraListenPorts; i++ {
				ports = append(ports, 7000+i)
			}
		})

		It("allows the user to configure policies", func(done Done) {
			By("checking that all test app instances have registered themselves")
			checkRegistry(appRegistry, 10*time.Second, 500*time.Millisecond, len(appsTest)*appInstances)

			appIPs := getAppIPs(appRegistry)
			sample := sampleIPs(appIPs, sampleSize)

			By(fmt.Sprintf("checking that the connection fails sampling %d out of %d IPs on %d ports", len(sample), len(appIPs), len(ports)))
			runWithTimeout("check connection failures", Timeout_Check, func() {
				assertConnectionFails(appProxy, sample, ports, proxyInstances)
			})

			By(fmt.Sprintf("creating %d policies", len(appsTest)*len(ports)))
			doAllPolicies("create", appProxy, appsTest, ports)

			By(fmt.Sprintf("waiting %s for policies to be updated on cells", Policy_Update_Wait))
			time.Sleep(Policy_Update_Wait)

			sample = sampleIPs(appIPs, sampleSize)
			By(fmt.Sprintf("checking that the connection succeeds sampling %d out of %d IPs on %d ports", len(sample), len(appIPs), len(ports)))
			runWithTimeout("check connection success", Timeout_Check, func() {
				assertConnectionSucceeds(appProxy, sample, ports, proxyInstances)
			})

			By("dumping stats to commit to stats repo")
			dumpStats(appProxy, config.AppsDomain)

			By("sleeping for 30 seconds while policies exist")
			time.Sleep(30 * time.Second)

			By(fmt.Sprintf("deleting %d policies", len(appsTest)*len(ports)))
			doAllPolicies("delete", appProxy, appsTest, ports)

			By(fmt.Sprintf("waiting %s for policies to be updated on cells", Policy_Update_Wait))
			time.Sleep(Policy_Update_Wait)

			sample = sampleIPs(appIPs, sampleSize)
			By(fmt.Sprintf("checking that the connection succeeds sampling %d out of %d IPs on %d ports", len(sample), len(appIPs), len(ports)))
			runWithTimeout("check connection failures, again", Timeout_Check, func() {
				assertConnectionFails(appProxy, sample, ports, proxyInstances)
			})

			close(done)
		}, 30*60 /* <-- overall spec timeout in seconds */)
	})
	Describe("sampleIPs", func() {
		var population []string
		BeforeEach(func() {
			population = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
		})

		It("returns a sample of unique choices from the population", func() {
			sample := sampleIPs(population, 9)
			Expect(len(sample)).To(Equal(9))
			for i := 0; i < len(sample); i++ {
				for j := i + 1; j < len(sample); j++ {
					Expect(sample[i]).NotTo(Equal(sample[j]))
				}
			}
		})
		Context("when the sample size is larger than the population", func() {
			It("returns the whole population", func() {
				sample := sampleIPs(population, 999)
				Expect(sample).To(Equal(population))
			})
		})
		Context("when the sample size is equal to the population size", func() {
			It("returns the whole population", func() {
				sample := sampleIPs(population, len(population))
				Expect(sample).To(Equal(population))
			})
		})
		Context("when the sample size is zero", func() {
			It("returns the whole population", func() {
				sample := sampleIPs(population, 0)
				Expect(sample).To(Equal(population))
			})
		})
		Context("when the sample size is negative", func() {
			It("returns the whole population", func() {
				sample := sampleIPs(population, -1)
				Expect(sample).To(Equal(population))
			})
		})
	})
})

func getToken() string {
	cmd := exec.Command("cf", "oauth-token")
	session, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session.Wait(Timeout_Short)).Should(gexec.Exit(0))
	rawOutput := string(session.Out.Contents())
	return strings.TrimSpace(strings.TrimPrefix(rawOutput, "bearer "))
}

func getGuids(sourceAppName string, dstAppNames []string) (string, []string) {
	dstGuids := []string{}
	sourceGuid := ""
	token := getToken()
	appsClient := rainmaker.NewApplicationsService(rainmaker.Config{Host: "http://" + config.ApiEndpoint})

	appsList, err := appsClient.List(token)
	Expect(err).NotTo(HaveOccurred())

	for {
		for _, app := range appsList.Applications {
			if app.Name == sourceAppName {
				sourceGuid = app.GUID
				continue
			}
			for _, tickAppName := range dstAppNames {
				if app.Name == tickAppName {
					dstGuids = append(dstGuids, app.GUID)
					break
				}
			}
		}
		if appsList.HasNextPage() {
			appsList, err = appsList.Next(token)
			Expect(err).NotTo(HaveOccurred())
		} else {
			break
		}
	}

	Expect(sourceGuid).NotTo(BeEmpty())
	Expect(dstGuids).To(HaveLen(len(dstAppNames)))

	return sourceGuid, dstGuids
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func doAllPolicies(action string, source string, dstList []string, dstPorts []int) {
	policyClient := policy_client.NewExternal(lagertest.NewTestLogger("test"), &http.Client{}, "http://"+config.ApiEndpoint)
	sourceGuid, dstGuids := getGuids(source, dstList)
	policies := []models.Policy{}
	for _, dstGuid := range dstGuids {
		for _, port := range dstPorts {
			policies = append(policies, models.Policy{
				Source: models.Source{
					ID: sourceGuid,
				},
				Destination: models.Destination{
					ID:       dstGuid,
					Port:     port,
					Protocol: "tcp",
				},
			})
		}
	}
	token := getToken()
	if action == "create" {
		Expect(policyClient.AddPolicies(token, policies)).To(Succeed())
	} else if action == "delete" {
		for i := 0; i < len(policies); i += 100 {
			Expect(policyClient.DeletePolicies(token, policies[i:min(i+100, len(policies))])).To(Succeed())
		}
	}
}

func runWithTimeout(operation string, timeout time.Duration, work func()) {
	done := make(chan bool)
	go func() {
		defer func() { close(done) }()
		defer GinkgoRecover()

		By(fmt.Sprintf("starting %s\n", operation))
		work()
		By(fmt.Sprintf("completed %s\n", operation))
		done <- true
	}()

	select {
	case ok := <-done:
		if !ok {
			Fail("failure during " + operation)
		}
	case <-time.After(timeout):
		Fail("timeout on " + operation)
	}
}

func dumpStats(host, domain string) {
	resp, err := httpGetBytes(fmt.Sprintf("http://%s.%s/stats", host, domain))
	Expect(err).NotTo(HaveOccurred())

	netStatsFile := os.Getenv("NETWORK_STATS_FILE")
	if netStatsFile != "" {
		Expect(ioutil.WriteFile(netStatsFile, resp.Body, 0600)).To(Succeed())
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

func sampleIPs(population []string, sampleSize int) []string {
	populationSize := len(population)
	if len(population) <= sampleSize || sampleSize < 1 {
		return population
	}
	var sample = []string{}
	for i := 0; i < sampleSize; i++ {
		j := rand.Intn(populationSize)
		sample = append(sample, population[j])
		population = append(population[:j], population[j+1:]...)
		populationSize--
	}
	return sample
}

func assertConnectionSucceeds(sourceApp string, destApps []string, ports []int, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 50 * nProxies,
	}
	parallelRunner.RunOnSliceStrings(destApps, func(appIP string) {
		for _, port := range ports {
			assertSingleConnection(appIP, port, sourceApp, true)
		}
	})
}

func assertConnectionFails(sourceApp string, destApps []string, ports []int, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 50 * nProxies,
	}
	parallelRunner.RunOnSliceStrings(destApps, func(appIP string) {
		for _, port := range ports {
			assertSingleConnection(appIP, port, sourceApp, false)
		}
	})
}

func assertSingleConnection(destIP string, port int, sourceAppName string, shouldSucceed bool) {
	if shouldSucceed {
		assertResponseContains(destIP, port, sourceAppName, "application_name")
	} else {
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
