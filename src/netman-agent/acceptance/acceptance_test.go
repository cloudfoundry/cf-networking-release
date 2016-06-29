package acceptance_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"netman-agent/config"
	"os/exec"

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
			w.Write([]byte(fmt.Sprintf("POLICIES VERSION %d", serverCallCount)))
			return
		}
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(`you asked for a path that we do not mock`))
	}))
}

var _ = Describe("Acceptance", func() {
	var (
		session          *gexec.Session
		conf             config.Config
		mockPolicyServer *httptest.Server
	)

	BeforeEach(func() {
		mockPolicyServer = createMockPolicyServer()
		conf = config.Config{
			PolicyServerURL: mockPolicyServer.URL,
			PollInterval:    1,
		}
		configFilePath := WriteConfigFile(conf)

		netmanAgentCmd := exec.Command(netmanAgentPath, "-config-file", configFilePath)
		var err error
		session, err = gexec.Start(netmanAgentCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

		if mockPolicyServer != nil {
			mockPolicyServer.Close()
			mockPolicyServer = nil
		}
	})

	Describe("boring daemon behavior", func() {
		It("should boot and gracefully terminate", func() {
			Consistently(session).ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})
	})

	It("polls the policy server on a regular interval", func() {
		Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*got-policies.*POLICIES VERSION 1`))
		Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*got-policies.*POLICIES VERSION 2`))
		Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*got-policies.*POLICIES VERSION 3`))
	})

	Context("if the policy server is unavailable", func() {
		BeforeEach(func() {
			mockPolicyServer.Close()
			mockPolicyServer = nil
		})
		It("should log", func() {
			Eventually(session.Out).Should(gbytes.Say(`.*server-error`))
		})
	})

	Context("if the policy server responds with an error status code", func() {
		BeforeEach(func() {
			mockPolicyServerResponseCode = 409
		})
		It("should log", func() {
			Eventually(session.Out, DEFAULT_TIMEOUT).Should(gbytes.Say(`.*policy-server-error.*409.*POLICIES VERSION 1`))
		})
	})
})
