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
		if numberFakeServerHits == 0 {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(400)
		}
		numberFakeServerHits++
	}))

	fakeSilkDaemonServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if numberSilkDaemonHits == 0 {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(400)
		}
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

		Expect(numberSilkDaemonHits).To(Equal(2))
	})

	It("pings the rep", func() {
		session := runTeardown(fakeRepServer.URL, fakeSilkDaemonServer.URL)
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		Expect(numberFakeServerHits).To(Equal(2))
	})

	It("pings the rep until the rep exits", func() {
		session := runTeardown(fakeRepServer.URL, fakeSilkDaemonServer.URL)
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		Expect(numberFakeServerHits).To(Equal(2))
		Expect(session.Out).To(gbytes.Say("waiting for the rep to exit"))

	})

	It("pings the silk daemon until it exits", func() {
		session := runTeardown(fakeRepServer.URL, fakeSilkDaemonServer.URL)
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		Expect(numberSilkDaemonHits).To(Equal(2))
		Expect(session.Out).To(gbytes.Say("waiting for the silk daemon to exit"))
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

	Context("When silk daemon will not exit", func() {
		BeforeEach(func() {
			fakeSilkDaemonServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(200)
				numberSilkDaemonHits++
			}))
		})

		It("pings the silk daemon server 5 times and fails gracefully", func() {
			session := runTeardown(fakeRepServer.URL, fakeSilkDaemonServer.URL)
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(1))

			Expect(numberSilkDaemonHits).To(Equal(5))
			Expect(session.Err).To(gbytes.Say("silk-daemon-teardown: Silk Daemon Server did not exit after 5 ping attempts"))
		})
	})
})

func runTeardown(url, silkDaemonUrl string) *gexec.Session {
	startCmd := exec.Command(paths.TeardownBin, "--repUrl", url, "--silkDaemonUrl", silkDaemonUrl, "--repTimeout", "0", "--silkDaemonTimeout", "0")
	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	return session
}
