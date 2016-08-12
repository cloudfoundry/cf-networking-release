package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Push = 5 * time.Minute
const Timeout_Short = 10 * time.Second
const port = 8080

func getSubnet(ip string) string {
	return strings.Split(ip, ".")[2]
}

func isSameCell(sourceIP, destIP string) bool {
	return getSubnet(sourceIP) == getSubnet(destIP)
}

var _ = Describe("connectivity between containers on the overlay network", func() {
	Describe("networking policy", func() {
		var (
			appProxy  string
			appReflex string
			orgName   string
			spaceName string
		)

		BeforeEach(func() {
			appProxy = fmt.Sprintf("proxy-%d", rand.Int31())
			appReflex = fmt.Sprintf("reflex-%d", rand.Int31())

			Auth(testConfig.TestUser, testConfig.TestUserPassword)

			orgName = "test-org"
			Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

			spaceName = "test-space"
			Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

			pushAppOfType(appProxy, "proxy")
			pushAppOfType(appReflex, "reflex")

			By("creating a new policy to allow the app to talk to itself")
			session := cf.Cf("access-allow", appReflex, appReflex, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(2 * Timeout_Short)
			Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

			scaleApp(appReflex, 4 /* instances */)
		})

		AfterEach(func() {
			appReport(appProxy, Timeout_Short)
			appReport(appReflex, Timeout_Short)

			// clean up everything
			Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		})

		It("allows the user to configure connections", func(done Done) {
			getPeers := func() ([]string, error) {
				resp, err := http.Get(fmt.Sprintf("http://%s.%s/peers", appReflex, config.AppsDomain))
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

			By("checking that the reflex app has discovered all its instances")
			Eventually(getPeers, 60*time.Second, 500*time.Millisecond).Should(HaveLen(4))

			By("checking that the connection fails")
			assertConnectionFails(appProxy, appReflex)

			By("creating a new policy")
			session := cf.Cf("access-allow", appProxy, appReflex, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(2 * Timeout_Short)
			Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

			By(fmt.Sprintf("checking that %s can reach %s", appProxy, appReflex))
			assertConnectionSucceeds(appProxy, appReflex)

			By("deleting the policy")
			session = cf.Cf("access-deny", appProxy, appReflex, "--protocol", "tcp", "--port", fmt.Sprintf("%d", port)).Wait(2 * Timeout_Short)
			Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))

			By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appReflex))
			assertConnectionFails(appProxy, appReflex)

			By("checking that reflex no longer reports deleted instances")
			scaleApp(appReflex, 1 /* instances */)
			Eventually(getPeers, 60*time.Second, 500*time.Millisecond).Should(HaveLen(1))

			close(done)
		}, 300 /* <-- overall spec timeout in seconds */)
	})
})

func assertConnectionSucceeds(sourceApp, destApp string) {
	assertAllConnectionStatus(sourceApp, destApp, true)
}

func assertConnectionFails(sourceApp, destApp string) {
	assertAllConnectionStatus(sourceApp, destApp, false)
}

func assertAllConnectionStatus(sourceApp, destApp string, shouldSucceed bool) {
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
		assertSingleConnection(sourceApp, destIP, shouldSucceed)
	}
}

func assertSingleConnection(sourceAppName string, destIP string, shouldSucceed bool) {
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
		Eventually(proxyTest).Should(ContainSubstring(destIP))
	} else {
		Eventually(proxyTest).Should(ContainSubstring("request failed"))
	}
}
