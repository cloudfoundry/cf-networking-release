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
	"os/exec"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"

	pusherConfig "cf-pusher/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/rainmaker"
)

const Timeout_Check = 20 * time.Minute

const Time_Format = "15:04:05"

var _ = Describe("how the container network performs at scale", func() {
	Describe("scaling tests", func() {
		var (
			proxyApps            []string
			tickApps             []string
			registryApp          string
			ports                []int
			policyClient         *policy_client.ExternalClient
			testConfig           pusherConfig.Config
			policyUpdateWaitTime time.Duration
		)
		BeforeEach(func() {

			testConfig = pushConfig
			registryApp = pushConfig.Prefix + "registry"
			policyUpdateWaitTime = time.Duration(testConfig.PolicyUpdateWaitSeconds) * time.Second

			logger := lager.NewLogger("test")
			logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))
			policyClient = policy_client.NewExternal(logger, &http.Client{}, "http://"+config.ApiEndpoint)

			By(fmt.Sprintf("%s checking that destination app instances have registered themselves", ts()))
			checkRegistry(registryApp, 10*time.Second, 500*time.Millisecond, pushConfig.Applications*pushConfig.AppInstances)
		})
		JustBeforeEach(func() {
			proxyApps = []string{}
			for i := 0; i < testConfig.ProxyApplications; i++ {
				proxyApps = append(proxyApps, fmt.Sprintf(testConfig.Prefix+"proxy-%d", i))
			}

			tickApps = []string{}
			for i := 0; i < testConfig.Applications; i++ {
				tickApps = append(tickApps, fmt.Sprintf(testConfig.Prefix+"tick-%d", i))
			}

			ports = []int{8080}
			for i := 0; i < testConfig.ExtraListenPorts; i++ {
				ports = append(ports, 7000+i)
			}
		})
		runScalingTest := func() {
			It("allows the user to configure policies", func(done Done) {
				By(fmt.Sprintf("%s testing with %d source apps and %d destination apps listening on %d ports", ts(), testConfig.ProxyApplications, testConfig.Applications, len(ports)))
				appIPs := getAppIPs(registryApp, tickApps)
				conns := connections(proxyApps, appIPs, ports)
				sample := sampleConnections(conns, testConfig.SampleSize)

				By(fmt.Sprintf("%s checking that the connection fails sampling %d out of %d connections", ts(), len(sample), len(conns)))
				runWithTimeout("check connection failures", Timeout_Check, func() {
					assertConnectionFails(sample, testConfig.ProxyInstances)
				})

				By(fmt.Sprintf("%s creating %d policies", ts(), len(proxyApps)*len(tickApps)*len(ports)))
				policies := getPolicies(proxyApps, tickApps, ports)
				Expect(policyClient.AddPolicies(getToken(), policies)).To(Succeed())

				By(fmt.Sprintf("%s waiting %s for policies to be updated on cells", ts(), policyUpdateWaitTime))
				time.Sleep(policyUpdateWaitTime)

				sample = sampleConnections(conns, testConfig.SampleSize)
				By(fmt.Sprintf("%s checking that the connection succeeds sampling %d out of %d connections", ts(), len(sample), len(conns)))
				runWithTimeout("check connection success", Timeout_Check, func() {
					assertConnectionSucceeds(sample, testConfig.ProxyInstances)
				})

				By(fmt.Sprintf("%s sleeping for 30 seconds while policies exist", ts()))
				time.Sleep(30 * time.Second)

				By(fmt.Sprintf("%s deleting %d policies", ts(), len(proxyApps)*len(tickApps)*len(ports)))
				Expect(policyClient.DeletePolicies(getToken(), policies)).To(Succeed())

				By(fmt.Sprintf("%s waiting %s for policies to be updated on cells", ts(), policyUpdateWaitTime))
				time.Sleep(policyUpdateWaitTime)

				sample = sampleConnections(conns, testConfig.SampleSize)
				By(fmt.Sprintf("%s checking that the connection fails sampling %d out of %d connections", ts(), len(sample), len(conns)))
				runWithTimeout("check connection failures, again", Timeout_Check, func() {
					assertConnectionFails(sample, testConfig.ProxyInstances)
				})
				close(done)
			}, 30*60) // 30 minutes
		}
		Context("when one client with many backends", func() {
			BeforeEach(func() {
				testConfig.ProxyApplications = 1
			})
			runScalingTest()
		})
		Context("when one server with many clients", func() {
			BeforeEach(func() {
				testConfig.Applications = 1
			})
			runScalingTest()
		})
	})

	Describe("sampleConnections", func() {
		var population []Connection
		BeforeEach(func() {
			population = []Connection{
				{Source: "a", Dest: "da", Port: 1},
				{Source: "b", Dest: "db", Port: 2},
				{Source: "c", Dest: "dc", Port: 3},
				{Source: "d", Dest: "dd", Port: 4},
				{Source: "e", Dest: "de", Port: 5},
				{Source: "f", Dest: "df", Port: 6},
				{Source: "g", Dest: "dg", Port: 7},
				{Source: "h", Dest: "dh", Port: 8},
				{Source: "i", Dest: "di", Port: 9},
				{Source: "j", Dest: "dj", Port: 0},
			}
		})

		It("returns a sample of unique choices from the population", func() {
			sample := sampleConnections(population, 9)
			Expect(len(sample)).To(Equal(9))
			for i := 0; i < len(sample); i++ {
				for j := i + 1; j < len(sample); j++ {
					Expect(sample[i]).NotTo(Equal(sample[j]))
				}
			}
		})
		Context("when the sample size is larger than the population", func() {
			It("returns the whole population", func() {
				sample := sampleConnections(population, 999)
				Expect(sample).To(Equal(population))
			})
		})
		Context("when the sample size is equal to the population size", func() {
			It("returns the whole population", func() {
				sample := sampleConnections(population, len(population))
				Expect(sample).To(Equal(population))
			})
		})
		Context("when the sample size is zero", func() {
			It("returns the whole population", func() {
				sample := sampleConnections(population, 0)
				Expect(sample).To(Equal(population))
			})
		})
		Context("when the sample size is negative", func() {
			It("returns the whole population", func() {
				sample := sampleConnections(population, -1)
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

func getGuids(srcAppNames, dstAppNames []string) ([]string, []string) {
	srcGuids := []string{}
	dstGuids := []string{}
	token := getToken()
	appsClient := rainmaker.NewApplicationsService(rainmaker.Config{Host: "http://" + config.ApiEndpoint})

	appsList, err := appsClient.List(token)
	Expect(err).NotTo(HaveOccurred())

	for {
		for _, app := range appsList.Applications {
			for _, proxyAppName := range srcAppNames {
				if app.Name == proxyAppName {
					srcGuids = append(srcGuids, app.GUID)
					break
				}
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

	Expect(srcGuids).To(HaveLen(len(srcAppNames)))
	Expect(dstGuids).To(HaveLen(len(dstAppNames)))

	return srcGuids, dstGuids
}

func getPolicies(srcList, dstList []string, dstPorts []int) []models.Policy {
	srcGuids, dstGuids := getGuids(srcList, dstList)
	policies := []models.Policy{}
	for _, srcGuid := range srcGuids {
		for _, dstGuid := range dstGuids {
			for _, port := range dstPorts {
				policies = append(policies, models.Policy{
					Source: models.Source{
						ID: srcGuid,
					},
					Destination: models.Destination{
						ID:       dstGuid,
						Port:     port,
						Protocol: "tcp",
					},
				})
			}
		}
	}
	return policies
}

func runWithTimeout(operation string, timeout time.Duration, work func()) {
	done := make(chan bool)
	go func() {
		defer func() { close(done) }()
		defer GinkgoRecover()

		By(fmt.Sprintf("%s starting %s\n", ts(), operation))
		work()
		By(fmt.Sprintf("%s completed %s\n", ts(), operation))
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

func getAppIPs(registry string, appNames []string) []string {
	instancesResponse, err := getInstancesFromA8(registry)
	Expect(err).NotTo(HaveOccurred())

	ips := []string{}
	for _, instance := range instancesResponse.Instances {
		for _, name := range appNames {
			if name == instance.ServiceName {
				ip, _, err := net.SplitHostPort(instance.Endpoint.Value)
				Expect(err).NotTo(HaveOccurred())
				ips = append(ips, ip)
			}
		}
	}
	return ips
}

func sampleConnections(population []Connection, sampleSize int) []Connection {
	populationSize := len(population)
	if len(population) <= sampleSize || sampleSize < 1 {
		return population
	}
	var sample = []Connection{}
	for i := 0; i < sampleSize; i++ {
		j := rand.Intn(populationSize)
		sample = append(sample, population[j])
		population = append(population[:j], population[j+1:]...)
		populationSize--
	}
	return sample
}

type Connection struct {
	Source string
	Dest   string
	Port   int
}

func connections(sourceApps, destApps []string, ports []int) []Connection {
	var conns []Connection
	for _, s := range sourceApps {
		for _, d := range destApps {
			for _, p := range ports {
				conns = append(conns, Connection{Source: s, Dest: d, Port: p})
			}
		}
	}
	return conns
}

func slice(conns []Connection) []interface{} {
	var s []interface{}
	for _, c := range conns {
		s = append(s, c)
	}
	return s
}

func assertConnectionSucceeds(conns []Connection, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 10 * nProxies,
	}
	parallelRunner.RunOnSlice(slice(conns), func(obj interface{}) {
		conn := obj.(Connection)
		assertResponseContains(conn.Dest, conn.Port, conn.Source, "application_name")
	})
}

func assertConnectionFails(conns []Connection, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 10 * nProxies,
	}
	parallelRunner.RunOnSlice(slice(conns), func(obj interface{}) {
		conn := obj.(Connection)
		assertResponseContains(conn.Dest, conn.Port, conn.Source, "request failed")
	})
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

func ts() string {
	return time.Now().Format(Time_Format)
}
