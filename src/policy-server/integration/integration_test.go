package integration_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os/exec"
	"policy-server/config"
	"strings"
	"time"

	"code.cloudfoundry.org/go-db-helpers/db"
	"code.cloudfoundry.org/go-db-helpers/metrics"
	"code.cloudfoundry.org/go-db-helpers/testsupport"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Integration", func() {

	Context("with a database", func() {
		var (
			session      *gexec.Session
			sessions     []*gexec.Session
			conf         config.Config
			address      string
			debugAddress string
			testDatabase *testsupport.TestDatabase

			fakeMetron metrics.FakeMetron
		)

		BeforeEach(func() {
			fakeMetron = metrics.NewFakeMetron()

			dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
			dbConnectionInfo := testsupport.GetDBConnectionInfo()
			testDatabase = dbConnectionInfo.CreateDatabase(dbName)

			template := DefaultTestConfig(testDatabase.DBConfig(), fakeMetron.Address())
			policyServerConfs := configurePolicyServers(template, 1)
			sessions = startPolicyServers(policyServerConfs)
			session = sessions[0]
			conf = policyServerConfs[0]

			address = fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
			debugAddress = fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort)
		})

		AfterEach(func() {
			stopPolicyServers(sessions)

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

				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("UptimeRequestTime"),
				))
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

				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
					HaveName("WhoAmIRequestTime"),
				))
			})

			It("has a log level thats configurable at runtime", func() {
				resp := makeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(ContainSubstring("some-user"))

				Expect(session.Out).To(gbytes.Say("container-networking.policy-server"))
				Expect(session.Out).NotTo(gbytes.Say("request made to whoami endpoint"))

				_ = makeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/log-level", conf.DebugServerHost, conf.DebugServerPort),
					strings.NewReader("debug"),
				)

				resp = makeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(ContainSubstring("some-user"))

				Expect(session.Out).To(gbytes.Say("container-networking.policy-server.*request made to policy-server"))
			})

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
		)

		BeforeEach(func() {
			dbConfig := db.Config{
				Type:             "postgres",
				ConnectionString: "postgres://:@1.2.3.4:9999/?sslmode=disable",
			}
			conf := DefaultTestConfig(dbConfig, "some-address")
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
