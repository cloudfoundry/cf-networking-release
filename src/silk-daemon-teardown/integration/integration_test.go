package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"net/http"
	"net/http/httptest"
	"github.com/onsi/gomega/gbytes"
)

var (
	DEFAULT_TIMEOUT = "5s"

)

var fakeRepServer *httptest.Server
var fakeSilkDaemonServer *httptest.Server
var numberFakeServerHits int
var numberSilkDaemonHits int

var _ = BeforeEach(func() {
	fakeRepServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		numberFakeServerHits++
	}))

	fakeSilkDaemonServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		numberSilkDaemonHits++
	}))
})

var _ = AfterEach(func() {
	numberFakeServerHits = 0
	numberSilkDaemonHits = 0
})

var _ = Describe("Teardown", func() {

	It("pings the silk daemon", func() {
		session := runTeardown(fakeRepServer.URL, fakeSilkDaemonServer.URL)
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		Expect(numberSilkDaemonHits).To(Equal(1))
	})

	It("pings the rep", func() {
		session := runTeardown(fakeRepServer.URL, fakeSilkDaemonServer.URL)
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		Expect(numberFakeServerHits).To(Equal(1))
	})

	Context("when connecting to the rep fails", func() {
		It("returns an error", func() {
			session := runTeardown("some/bad/url", fakeSilkDaemonServer.URL)
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("silk-daemon-teardown: pinging rep failed with: Get some/bad/url: unsupported protocol scheme"))

		})
	})

	Context("when connecting to the silk daemon fails", func() {
		It("returns an error", func() {
			session := runTeardown(fakeRepServer.URL, "some/bad/url")
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("silk-daemon-teardown: pinging silk-daemon failed with: Get some/bad/url: unsupported protocol scheme"))

		})
	})

})

func runTeardown(url, silkDaemonUrl string) *gexec.Session {
	startCmd := exec.Command(paths.TeardownBin, "--repUrl", url, "--silkDaemonUrl", silkDaemonUrl)
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	return session
}
