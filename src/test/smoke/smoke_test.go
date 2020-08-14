package smoke_test

import (
	"cf-pusher/cf_cli_adapter"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Short = 10 * time.Second

var _ = Describe("connectivity between containers on the overlay network", func() {
	Describe("networking policy", func() {
		var (
			appProxy     string
			appSmoke     string
			appInstances int
			prefix       string
			spaceName    string
			orgName      string
			cfCli        *cf_cli_adapter.Adapter
		)

		BeforeEach(func() {
			prefix = config.Prefix

			if config.SmokeOrg == "" {
				orgName = prefix + "org"
				Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			} else {
				orgName = config.SmokeOrg
			}
			Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			spaceName = prefix + "inter-container-connectivity"
			Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

			Expect(cf.Cf("set-space-role", config.SmokeUser, orgName, spaceName, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))

			appInstances = config.AppInstances

			appProxy = prefix + "proxy"
			appSmoke = prefix + "smoke"

			cfCli = cf_cli_adapter.NewAdapter()
		})

		AfterEach(func() {
			appReport(appProxy, Timeout_Short)
			appReport(appSmoke, Timeout_Short)
			Expect(cf.Cf("delete-space", spaceName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		})

		It("allows the user to configure policies", func(done Done) {
			pushApp(appProxy, "proxy")
			pushApp(appSmoke, "smoke", "--no-start")
			setEnv(appSmoke, "PROXY_APP_URL", fmt.Sprintf("http://%s.%s", appProxy, config.AppsDomain))
			start(appSmoke)

			scaleApp(appSmoke, appInstances)

			ports := []int{8080}
			appsSmoke := []string{appSmoke}

			By("checking that the connection fails")
			runWithTimeout("check connection failures", 5*time.Minute, func() {
				assertConnectionFails(appSmoke, appInstances)
			})

			By("creating policies")
			createAllPolicies(appProxy, appsSmoke, ports, cfCli)

			// we should wait for minimum (pollInterval * 2)
			By("waiting for policies to be created on cells")
			time.Sleep(10 * time.Second)

			By(fmt.Sprintf("checking that %s can reach %s", appProxy, appsSmoke))
			runWithTimeout("check connection success", 5*time.Minute, func() {
				assertConnectionSucceeds(appSmoke, appInstances)
			})

			By("deleting policies")
			deleteAllPolicies(appProxy, appsSmoke, ports, cfCli)

			By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsSmoke))
			runWithTimeout("check connection failures, again", 5*time.Minute, func() {
				assertConnectionFails(appSmoke, appInstances)
			})

			close(done)
		}, 30*60 /* <-- overall spec timeout in seconds */)
	})
})

func createAllPolicies(sourceApp string, dstList []string, dstPorts []int, cfCli *cf_cli_adapter.Adapter) {
	for _, destApp := range dstList {
		for _, port := range dstPorts {
			err := cfCli.AddNetworkPolicy(sourceApp, destApp, port, "tcp")
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func deleteAllPolicies(sourceApp string, dstList []string, dstPorts []int, cfCli *cf_cli_adapter.Adapter) {
	for _, destApp := range dstList {
		for _, port := range dstPorts {
			err := cfCli.RemoveNetworkPolicy(sourceApp, destApp, port, "tcp")
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func runWithTimeout(operation string, timeout time.Duration, work func()) {
	done := make(chan bool)
	go func() {
		defer GinkgoRecover()
		fmt.Printf("starting %s\n", operation)
		work()
		done <- true
	}()

	select {
	case <-done:
		fmt.Printf("completed %s\n", operation)
		return
	case <-time.After(timeout):
		Fail("timeout on " + operation)
	}
}

func assertConnectionSucceeds(sourceApp string, appInstances int) {
	for i := 0; i < appInstances; i++ {
		assertSingleConnection(sourceApp, true)
	}
}

func assertConnectionFails(sourceApp string, appInstances int) {
	for i := 0; i < appInstances; i++ {
		assertSingleConnection(sourceApp, false)
	}
}

func assertSingleConnection(sourceAppName string, shouldSucceed bool) {
	if shouldSucceed {
		By("eventually smoke should reach itself")
		assertResponseContains(sourceAppName, "OK")
	} else {
		By("eventually smoke should NOT reach itself")
		assertResponseContains(sourceAppName, "FAILED")
	}
}

func assertResponseContains(sourceAppName string, desiredResponse string) {
	proxyTest := func() (string, error) {
		resp, err := httpGetBytes(fmt.Sprintf("http://%s.%s/selfproxy", sourceAppName, config.AppsDomain))
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
