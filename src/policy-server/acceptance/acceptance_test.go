package acceptance_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"lib/db"
	"lib/testsupport"
	"math/rand"
	"net/http"
	"netmon/acceptance/fakes"
	"os/exec"
	"policy-server/config"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Integration", func() {
	Context("with a database", func() {
		var (
			session      *gexec.Session
			conf         config.Config
			address      string
			testDatabase *testsupport.TestDatabase

			fakeMetron fakes.FakeMetron
		)

		var serverIsAvailable = func() error {
			return VerifyTCPConnection(address)
		}

		BeforeEach(func() {
			fakeMetron = fakes.New()

			dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
			dbConnectionInfo := testsupport.GetDBConnectionInfo()
			testDatabase = dbConnectionInfo.CreateDatabase(dbName)

			conf = config.Config{
				ListenHost:      "127.0.0.1",
				ListenPort:      9001 + GinkgoParallelNode(),
				UAAClient:       "test",
				UAAClientSecret: "test",
				UAAURL:          mockUAAServer.URL,
				Database:        testDatabase.DBConfig(),
				TagLength:       1,
				MetronAddress:   fakeMetron.Address(),
			}
			configFilePath := WriteConfigFile(conf)

			policyServerCmd := exec.Command(policyServerPath, "-config-file", configFilePath)
			var err error
			session, err = gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			address = fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)

			Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())

			if testDatabase != nil {
				testDatabase.Destroy()
			}

			Expect(fakeMetron.Close()).To(Succeed())
		})

		Describe("boring server behavior", func() {
			It("should boot and gracefully terminate", func() {
				Consistently(session).ShouldNot(gexec.Exit())

				session.Interrupt()
				Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
			})

			It("responds with uptime when accessed on the root path", func() {
				req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/", conf.ListenHost, conf.ListenPort), nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(responseString).To(ContainSubstring("Network policy server, up for"))
			})

			It("responds with uptime when accessed on the context path", func() {
				req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/networking", conf.ListenHost, conf.ListenPort), nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(responseString).To(ContainSubstring("Network policy server, up for"))
			})

			It("has a whoami endpoint", func() {
				resp := makeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(ContainSubstring("some-user"))
			})

			var HaveName = func(name string) types.GomegaMatcher {
				return WithTransform(func(ev fakes.Event) string {
					return ev.Name
				}, Equal(name))
			}

			It("should emit some metrics", func() {
				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(
						HaveName("uptime"),
					))

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(
						HaveName("totalPolicies"),
					))
			})
		})
	})

	Context("when the database is down", func() {
		var (
			session *gexec.Session
			conf    config.Config
		)

		BeforeEach(func() {
			conf = config.Config{
				ListenHost:      "127.0.0.1",
				ListenPort:      9001 + GinkgoParallelNode(),
				UAAClient:       "test",
				UAAClientSecret: "test",
				UAAURL:          mockUAAServer.URL,
				Database: db.Config{
					Type:     "postgres",
					Host:     "1.2.3.4",
					Port:     9999,
					Username: "",
					Password: "",
					Name:     "",
					SSLMode:  "",
				},
				TagLength: 1,
			}
			configFilePath := WriteConfigFile(conf)

			policyServerCmd := exec.Command(policyServerPath, "-config-file", configFilePath)
			var err error
			session, err = gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		It("should log and exit after 5 seconds", func() {
			Eventually(session, 90*time.Second).Should(gexec.Exit())

			Expect(session.Err).To(gbytes.Say("db connection timeout"))
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
