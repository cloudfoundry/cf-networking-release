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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var mockPolicyServerResponseCode int = 200

func createMockPolicyServer() *httptest.Server {
	var serverCallCount = 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/networking/v0/internal/policies" {
			serverCallCount += 1
			w.WriteHeader(mockPolicyServerResponseCode)
			w.Write([]byte(fmt.Sprintf(`{ "policies": [{"source": {"id": "app-%d"}, "destination": { "id": "other-app", "port": 8080, "protocol": "tcp"}}]}`, serverCallCount)))
			return
		}
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(fmt.Sprintf(`you asked for a path that we do not mock %s`, r.URL.Path)))
	}))
}

var _ = Describe("Acceptance", func() {
	var (
		session          *gexec.Session
		conf             config.Config
		mockPolicyServer *httptest.Server
		address          string
	)

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	BeforeEach(func() {
		mockPolicyServer = createMockPolicyServer()
		listenPort := 6666 + GinkgoParallelNode() + rand.Intn(5000)
		address = fmt.Sprintf("127.0.0.1:%d", listenPort)
		conf = config.Config{
			PolicyServerURL: mockPolicyServer.URL,
			PollInterval:    1,
			ListenHost:      "127.0.0.1",
			ListenPort:      listenPort,
		}
		configFilePath := WriteConfigFile(conf)

		netmanAgentCmd := exec.Command(netmanAgentPath, "-config-file", configFilePath)
		var err error
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
				mockPolicyServerResponseCode = 409
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
