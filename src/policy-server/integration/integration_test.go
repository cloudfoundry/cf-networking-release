package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"policy-server/config"
	"policy-server/integration/helpers"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

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
			address           string
			debugAddress      string
			dbConf            db.Config
			headers           map[string]string
			policyServerConfs []config.Config
			fakeMetron        metrics.FakeMetron
		)

		BeforeEach(func() {
			fakeMetron = metrics.NewFakeMetron()

			dbConf = testsupport.GetDBConfig()
			dbConf.DatabaseName = fmt.Sprintf("integration_test_node_%d", ports.PickAPort())

			template, _ := helpers.DefaultTestConfig(dbConf, fakeMetron.Address(), "fixtures")
			policyServerConfs = configurePolicyServers(template, 1)
			sessions = startPolicyServers(policyServerConfs)
			session = sessions[0]
			conf = policyServerConfs[0]

			address = fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
			debugAddress = fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort)
		})

		AfterEach(func() {
			stopPolicyServers(sessions, policyServerConfs, nil)

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
			badDbConfig := db.Config{
				Type:         "postgres",
				User:         "invalidUser",
				Password:     "badPassword",
				Host:         "badHost",
				Port:         9999,
				DatabaseName: "nonexistentDatabase",
				Timeout:      5,
			}
			conf, _ := helpers.DefaultTestConfig(badDbConfig, "some-address", "fixtures")
			configFilePath := helpers.WriteConfigFile(conf)

			policyServerCmd := exec.Command(policyServerPath, "-config-file", configFilePath)
			var err error
			session, err = gexec.Start(policyServerCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			session.Interrupt()
			Eventually(session, helpers.DEFAULT_TIMEOUT).Should(gexec.Exit())
		})

		It("should log and exit after 5 seconds", func() {
			Eventually(session, 90*time.Second).Should(gexec.Exit())

			Expect(session.Err).To(gbytes.Say("testprefix.policy-server: db connection timeout"))
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
				conf, _ := helpers.DefaultTestConfig(badDbConfig, "some-address", "fixtures")
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
