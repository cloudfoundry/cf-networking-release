package integration_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Integration", func() {

	Context("with a database", func() {
		var (
			session           *gexec.Session
			sessions          []*gexec.Session
			conf              config.Config
			dbConf            db.Config
			headers           map[string]string
			policyServerConfs []config.Config
			fakeMetron        metrics.FakeMetron
		)

		BeforeEach(func() {
			fakeMetron = metrics.NewFakeMetron()

			dbConf = testsupport.GetDBConfig()
			dbConf.DatabaseName = fmt.Sprintf("integration_test_node_%d", ports.PickAPort())

			template, _, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
			policyServerConfs = configurePolicyServers(template, 1)
			conf = policyServerConfs[0]
		})

		JustBeforeEach(func() {
			sessions = startPolicyServers(policyServerConfs)
			session = sessions[0]
		})

		AfterEach(func() {
			stopPolicyServers(sessions, policyServerConfs)
			Expect(fakeMetron.Close()).To(Succeed())
		})

		Describe("boring server behavior", func() {
			It("should boot and gracefully terminate", func() {
				Consistently(session).ShouldNot(gexec.Exit())

				session.Interrupt()
				Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())
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
				resp := helpers.MakeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
					headers,
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

			Context("whoami endpoint with a client", func() {
				It("responds with the client id", func() {
					clientAuthHeaders := map[string]string{
						"Authorization": "Bearer valid-client-token",
					}
					resp := helpers.MakeAndDoRequest(
						"GET",
						fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
						clientAuthHeaders,
						nil,
					)

					Expect(resp.StatusCode).To(Equal(http.StatusOK))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(responseString).To(ContainSubstring("some-client-id"))

					Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
						HaveName("WhoAmIRequestTime"),
					))
				})
			})

			Context("with enabled TLS", func() {
				var (
					tlsConfig *tls.Config
				)

				BeforeEach(func() {
					tlsConfig = helpers.DefaultTLSConfig()

					for i := range policyServerConfs {
						policyServerConfs[i].EnableTLS = true
					}
				})

				It("responds to whoami endpoint", func() {
					resp := helpers.MakeAndDoHTTPSRequest(
						"GET",
						fmt.Sprintf("https://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
						nil,
						tlsConfig,
					)

					Expect(resp.StatusCode).To(Equal(http.StatusOK))
					responseString, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(responseString).To(ContainSubstring("some-user"))

					Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(
						HaveName("WhoAmIRequestTime"),
					))
				})
			})

			It("has a log level thats configurable at runtime", func() {
				resp := helpers.MakeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
					headers,
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(ContainSubstring("some-user"))

				Expect(session.Out).To(gbytes.Say("testprefix.policy-server"))
				Expect(session.Out).NotTo(gbytes.Say("request made to whoami endpoint"))

				_ = helpers.MakeAndDoRequest(
					"POST",
					fmt.Sprintf("http://%s:%d/log-level", conf.DebugServerHost, conf.DebugServerPort),
					headers,
					strings.NewReader("debug"),
				)

				resp = helpers.MakeAndDoRequest(
					"GET",
					fmt.Sprintf("http://%s:%d/networking/v0/external/whoami", conf.ListenHost, conf.ListenPort),
					headers,
					nil,
				)

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				responseString, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseString).To(ContainSubstring("some-user"))

				Expect(session.Out).To(gbytes.Say("testprefix.policy-server.request_.*serving"))
				Expect(session.Out).To(gbytes.Say("testprefix.policy-server.request_.*done"))
			})

			It("should emit some metrics", func() {
				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "uptime")),
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "totalPolicies")),
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "DBOpenConnections")),
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "DBQueriesTotal")),
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "DBQueriesSucceeded")),
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "DBQueriesFailed")),
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "DBQueriesInFlight")),
				)

				Eventually(fakeMetron.AllEvents, "5s").Should(
					ContainElement(HaveOriginAndName("policy-server", "DBQueryDurationMax")),
				)
			})
		})
	})

	Context("when connection to the database times out", func() {
		var (
			session *gexec.Session
		)

		BeforeEach(func() {
			badDbConfig := db.Config{
				Type:         "postgres",
				User:         "invalidUser",
				Password:     "badPassword",
				Host:         "badHost",
				Port:         9999,
				DatabaseName: "nonexistentDatabase",
				Timeout:      1,
			}
			conf, _, _ := helpers.DefaultTestConfig(badDbConfig, "some-address", "fixtures")
			configFilePath := helpers.WriteConfigFile(conf)

			policyServerCmd := exec.Command(policyServerPath, "-config-file", configFilePath)
			var err error
			session, err = gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session.Out).Should(gbytes.Say("getting db connection"))
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		It("should log and exit with a timeout error", func() {
			Eventually(session, 10*time.Second).Should(gexec.Exit())
			Expect(session.Err).To(gbytes.Say("testprefix.policy-server: db connect: unable to ping: context deadline exceeded"))
		})
	})

	Describe("Config file errors", func() {
		var (
			session *gexec.Session
		)
		Context("when the config file is invalid", func() {
			BeforeEach(func() {
				badDbConfig := db.Config{
					Type:         "",
					User:         "",
					Password:     "",
					Host:         "",
					Port:         0,
					DatabaseName: "nonexistentDatabase",
					Timeout:      0,
				}
				conf, _, _ := helpers.DefaultTestConfig(badDbConfig, "some-address", "fixtures")
				configFilePath := helpers.WriteConfigFile(conf)

				policyServerCmd := exec.Command(policyServerPath, "-config-file", configFilePath)
				var err error
				session, err = gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

			})
			It("exits and errors", func() {
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("cfnetworking.policy-server: could not read config file: invalid config: "))
			})
		})
		Context("when the config file argument is not included", func() {
			BeforeEach(func() {
				policyServerCmd := exec.Command(policyServerPath)
				var err error
				session, err = gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
			})
			It("exits and errors", func() {
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("cfnetworking.policy-server: could not read config file: reading config: open"))
			})
		})
	})
})
