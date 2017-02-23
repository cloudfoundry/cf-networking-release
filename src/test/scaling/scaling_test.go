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

// 2 * poll cycle time (5s)
const Policy_Update_Wait = 10 * time.Second

const Time_Format = "15:04:05"

var _ = Describe("how the container network performs at scale", func() {
	Describe("scaling tests", func() {
		var (
			proxyApps   []string
			tickApps    []string
			registryApp string
			ports       []int
			testConfig  pusherConfig.Config
		)
		BeforeEach(func() {
			testConfig = pushConfig
			registryApp = pushConfig.Prefix + "registry"
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
				By(fmt.Sprintf("%s testing with %d source apps and %d destination apps", ts(), testConfig.ProxyApplications, testConfig.Applications))
				appIPs := getAppIPs(registryApp, tickApps)
				sample := sampleIPs(appIPs, testConfig.SampleSize)

				By(fmt.Sprintf("%s checking that the connection fails sampling %d out of %d IPs on %d ports", ts(), len(sample), len(appIPs), len(ports)))
				runWithTimeout("check connection failures", Timeout_Check, func() {
					assertConnectionFails(proxyApps, sample, ports, testConfig.ProxyInstances)
				})

				By(fmt.Sprintf("%s creating %d policies", ts(), len(proxyApps)*len(tickApps)*len(ports)))
				doAllPolicies("create", proxyApps, tickApps, ports)

				By(fmt.Sprintf("%s waiting %s for policies to be updated on cells", ts(), Policy_Update_Wait))
				time.Sleep(Policy_Update_Wait)

				sample = sampleIPs(appIPs, testConfig.SampleSize)
				By(fmt.Sprintf("%s checking that the connection succeeds sampling %d out of %d IPs on %d ports to proxy", ts(), len(sample), len(appIPs), len(ports)))
				runWithTimeout("check connection success", Timeout_Check, func() {
					assertConnectionSucceeds(proxyApps, sample, ports, testConfig.ProxyInstances)
				})

				By(fmt.Sprintf("%s sleeping for 30 seconds while policies exist", ts()))
				time.Sleep(30 * time.Second)

				By(fmt.Sprintf("%s deleting %d policies", ts(), len(proxyApps)*len(tickApps)*len(ports)))
				doAllPolicies("delete", proxyApps, tickApps, ports)

				By(fmt.Sprintf("%s waiting %s for policies to be updated on cells", ts(), Policy_Update_Wait))
				time.Sleep(Policy_Update_Wait)

				sample = sampleIPs(appIPs, testConfig.SampleSize)
				By(fmt.Sprintf("%s checking that the connection fails sampling %d out of %d IPs on %d ports", ts(), len(sample), len(appIPs), len(ports)))
				runWithTimeout("check connection failures, again", Timeout_Check, func() {
					assertConnectionFails(proxyApps, sample, ports, testConfig.ProxyInstances)
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

func doAllPolicies(action string, srcList, dstList []string, dstPorts []int) {

	logger := lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

	policyClient := policy_client.NewExternal(logger, &http.Client{}, "http://"+config.ApiEndpoint)
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
	token := getToken()
	if action == "create" {
		Expect(policyClient.AddPolicies(token, policies)).To(Succeed())
	} else if action == "delete" {
		Expect(policyClient.DeletePolicies(token, policies)).To(Succeed())
	}
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

type SrcDstPair struct {
	Source string
	Dest   string
}

func assertConnectionSucceeds(sourceApps, destApps []string, ports []int, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 50 * nProxies,
	}
	pairs := []interface{}{}
	for _, s := range sourceApps {
		for _, d := range destApps {
			pairs = append(pairs, SrcDstPair{Source: s, Dest: d})
		}
	}
	parallelRunner.RunOnSlice(pairs, func(obj interface{}) {
		pair := obj.(SrcDstPair)
		for _, port := range ports {
			assertSingleConnection(pair.Dest, port, pair.Source, true)
		}
	})
}

func assertConnectionFails(sourceApps, destApps []string, ports []int, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 50 * nProxies,
	}
	pairs := []interface{}{}
	for _, s := range sourceApps {
		for _, d := range destApps {
			pairs = append(pairs, SrcDstPair{Source: s, Dest: d})
		}
	}
	parallelRunner.RunOnSlice(pairs, func(obj interface{}) {
		pair := obj.(SrcDstPair)
		for _, port := range ports {
			assertSingleConnection(pair.Dest, port, pair.Source, false)
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

func ts() string {
	return time.Now().Format(Time_Format)
}
