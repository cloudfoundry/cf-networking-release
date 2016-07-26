package acceptance_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"netman-agent/config"
	"os/exec"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Acceptance", func() {
	var (
		session                      *gexec.Session
		conf                         config.Config
		mockPolicyServer             *httptest.Server
		address                      string
		mockPolicyServerResponseCode int
		mockServerControl            *sync.Mutex
	)

	// without a lock around it, we get race conditions when changing the policy server response code
	setResponseCode := func(code int) {
		mockServerControl.Lock()
		defer mockServerControl.Unlock()
		mockPolicyServerResponseCode = code
	}
	getResponseCode := func() int {
		mockServerControl.Lock()
		defer mockServerControl.Unlock()
		return mockPolicyServerResponseCode
	}

	createMockPolicyServer := func() *httptest.Server {
		var serverCallCount = 0
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/networking/v0/internal/policies" {
				serverCallCount += 1
				w.WriteHeader(getResponseCode())
				w.Write([]byte(fmt.Sprintf(`{
					"policies": [{
						"source": {"id": "app-%d", "tag": "BEEF"},
						"destination": { "id": "other-app", "port": 8080, "protocol": "tcp"}
					}]}`, serverCallCount)))
				return
			}
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte(fmt.Sprintf(`you asked for a path that we do not mock %s`, r.URL.Path)))
		}))
	}

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	BeforeEach(func() {
		subnetFile, err := ioutil.TempFile("", "subnet.env")
		Expect(err).NotTo(HaveOccurred())
		_, err = subnetFile.WriteString(`
			FLANNEL_NETWORK=10.255.0.0/16
			FLANNEL_SUBNET=10.255.19.1/24
			FLANNEL_MTU=1450
			FLANNEL_IPMASQ=false
		`)
		Expect(subnetFile.Close()).To(Succeed())
		Expect(err).NotTo(HaveOccurred())

		mockServerControl = &sync.Mutex{}
		setResponseCode(200)
		mockPolicyServer = createMockPolicyServer()
		listenPort := 6666 + GinkgoParallelNode() + rand.Intn(5000)
		address = fmt.Sprintf("127.0.0.1:%d", listenPort)
		conf = config.Config{
			PolicyServerURL:   mockPolicyServer.URL,
			PollInterval:      1,
			ListenHost:        "127.0.0.1",
			ListenPort:        listenPort,
			VNI:               42,
			FlannelSubnetFile: subnetFile.Name(),
		}
		configFilePath := WriteConfigFile(conf)

		netmanAgentCmd := exec.Command(netmanAgentPath, "-config-file", configFilePath)
		session, err = gexec.Start(netmanAgentCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		Eventually(serverIsAvailable, DEFAULT_TIMEOUT).ShouldNot(Succeed())

		if mockPolicyServer != nil {
			mockPolicyServer.Close()
			Eventually(func() error {
				return VerifyTCPConnection(
					strings.TrimPrefix("http://", mockPolicyServer.URL))
			}).ShouldNot(Succeed())
			mockPolicyServer = nil
		}
	})

	Describe("boring daemon and server behavior", func() {
		It("should boot and gracefully terminate", func() {
			Consistently(session).ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})
	})

	Describe("creating rules", func() {
		It("writes default deny rules", func() {
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`"chain":"netman--local-.*"-i","cni-flannel0","-m","state","--state","ESTABLISHED,RELATED","-j","ACCEPT".*"table":"filter"`))
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`"chain":"netman--local-.*"-i","cni-flannel0","-s","10.255.19.1/24","-d","10.255.19.1/24","-j","DROP".*"table":"filter"`))
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`"chain":"netman--remote-.*"-i","flannel.42","-m","state","--state","ESTABLISHED,RELATED","-j","ACCEPT".*"table":"filter"`))
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`"chain":"netman--remote-.*"-i","flannel.42","-j","DROP".*"table":"filter"`))
		})

		It("writes a rule to allow outbound access to the internet", func() {
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`"chain":"netman--postrout-.*"-s","10.255.19.1/24","!","-d","10.255.0.0/16","-j","MASQUERADE".*"table":"nat"`))
		})

		Context("when an app container comes up and has its results posted to cni_result", func() {
			It("gets rules based on what policies are configured on the server", func() {
				body := strings.NewReader(`{
							"container_id":  "some-container-id",
							"group_id":  "app-3",
							"ip":  "1.2.3.4"
						}`)
				_ = makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/cni_result", conf.ListenHost, conf.ListenPort),
					body,
				)
				body = strings.NewReader(`{
							"container_id":  "some-other-container-id",
							"group_id":  "other-app",
							"ip":  "5.6.7.8"
						}`)
				_ = makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/cni_result", conf.ListenHost, conf.ListenPort),
					body,
				)
				Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`properties":\["-i","cni-flannel0","-s","1.2.3.4","-d","5.6.7.8","-p","tcp","--dport","8080","-j","ACCEPT"\]`))
			})
		})
	})

	Describe("getting policy updates", func() {
		It("polls the policy server on a regular interval", func() {
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*get-policies.*app-1`))
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*get-policies.*app-2`))
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*get-policies.*app-3`))
		})

		Context("if the policy server is unavailable", func() {
			BeforeEach(func() {
				mockPolicyServer.Close()
				mockPolicyServer = nil
			})
			It("should log", func() {
				Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*get-policies.*error`))
			})
		})

		Context("if the policy server responds with an error status code", func() {
			BeforeEach(func() {
				setResponseCode(409)
			})
			It("should log", func() {
				Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`policy-client.http-client.*app-.*409`))
			})
		})
	})

	Describe("netman server", func() {
		It("logs post requests to /cni_result", func() {
			body := strings.NewReader(`{
				"container_id":  "some-container-id",
				"group_id":  "some-app-guid",
				"ip":  "1.2.3.4"
			}`)
			resp := makeAndDoRequest(
				"POST",
				fmt.Sprintf("http://%s:%d/cni_result", conf.ListenHost, conf.ListenPort),
				body,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))

			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*cni_result_add.*container_id.*some-container-id.*group_id.*some-app-guid.*ip.*1.2.3.4`))
		})

		It("logs delete requests to /cni_result", func() {
			body := strings.NewReader(`{
				"container_id":  "some-container-id"
			}`)
			resp := makeAndDoRequest(
				"DELETE",
				fmt.Sprintf("http://%s:%d/cni_result", conf.ListenHost, conf.ListenPort),
				body,
			)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			responseString, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseString).To(MatchJSON("{}"))

			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*cni_result_del.*container_id.*some-container-id`))
		})
	})
})

func makeAndDoRequest(method string, endpoint string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, endpoint, body)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}
