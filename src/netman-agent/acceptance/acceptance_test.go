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

var serverCallCount = 0
var mockPolicyServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/internal/v0/policies" {
		serverCallCount += 1
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("POLICIES VERSION %d", serverCallCount)))
		return
	}
	w.WriteHeader(http.StatusTeapot)
	w.Write([]byte(`you asked for a path that we do not mock`))
}))

var _ = Describe("Acceptance", func() {
	var (
		session *gexec.Session
		conf    config.Config
	)

	BeforeEach(func() {
		conf = config.Config{
			PolicyServerURL: mockPolicyServer.URL,
			PollInterval:    1,
		}
		configFilePath := WriteConfigFile(conf)
		serverCallCount = 0

		netmanAgentCmd := exec.Command(netmanAgentPath, "-config-file", configFilePath)
		var err error
		session, err = gexec.Start(netmanAgentCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
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
})
