package main_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"

	"fmt"
	"time"

	"crypto/tls"
	"test-helpers"

	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Service Discovery Controller process", func() {
	var (
		session                   *gexec.Session
		configPath                string
		port                      int
		natsServerPort            int
		natsServer                *server.Server
		routeEmitter              *nats.Conn
		clientCert                tls.Certificate
		caFile                    string
		serverCert                string
		serverKey                 string
		stalenessThresholdSeconds int
		pruningIntervalSeconds    int
		logLevelEndpointPort      int
		logLevelEndpointAddress   string
		fakeMetron                metrics.FakeMetron
	)

	BeforeEach(func() {
		caFile, serverCert, serverKey, clientCert = testhelpers.GenerateCaAndMutualTlsCerts()
		stalenessThresholdSeconds = 1
		pruningIntervalSeconds = 1
		logLevelEndpointPort = ports.PickAPort()
		logLevelEndpointAddress = "localhost"
		fakeMetron = metrics.NewFakeMetron()
		natsServerPort = ports.PickAPort()
		natsServer = RunNatsServerOnPort(natsServerPort)
		port = ports.PickAPort()
		configPath = writeConfigFile(fmt.Sprintf(`{
			"address":"127.0.0.1",
			"port":"%d",
			"ca_cert": "%s",
			"server_cert": "%s",
			"server_key": "%s",
			"nats":[
				{
					"host":"localhost",
					"port":%d,
					"user":"",
					"pass":""
				}
			],
			"staleness_threshold_seconds": %d,
			"pruning_interval_seconds": %d,
			"log_level_address": "%s",
			"log_level_port": %d,
			"metron_port": %d,
			"metrics_emit_seconds": 2,
			"resume_pruning_delay_seconds": 1,
			"warm_duration_seconds": 0
		}`,
			port, caFile, serverCert, serverKey, natsServerPort, stalenessThresholdSeconds, pruningIntervalSeconds, logLevelEndpointAddress, logLevelEndpointPort, fakeMetron.Port()))
	})

	AfterEach(func() {
		session.Kill()
		os.Remove(configPath)
		natsServer.Shutdown()
	})

	Context("when it starts successfully", func() {
		JustBeforeEach(func() {
			startCmd := exec.Command(pathToServer, "-c", configPath)
			var err error
			session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session, 6*time.Second).Should(gbytes.Say("service-discovery-controller.server-started"))

			routeEmitter = newFakeRouteEmitter("nats://" + natsServer.Addr().String())
			register(routeEmitter, "192.168.0.1", "app-id.internal.local.")
			register(routeEmitter, "192.168.0.2", "app-id.internal.local.")
			register(routeEmitter, "192.168.0.1", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.2", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.3", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.4", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.5", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.6", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.7", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.8", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.9", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.10", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.11", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.12", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.13", "large-id.internal.local.")
			Expect(routeEmitter.Flush()).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session.Kill()
			os.Remove(configPath)
			routeEmitter.Close()
			natsServer.Shutdown()
		})

		It("accepts interrupt signals and shuts down", func() {
			session.Signal(os.Interrupt)

			Eventually(session).Should(gexec.Exit())
			Eventually(session).Should(gbytes.Say("service-discovery-controller.server-stopped"))
		})

		It("should not return ips for unregistered domains", func() {
			requestLogChange("debug", logLevelEndpointPort)

			unregister(routeEmitter, "192.168.0.1", "app-id.internal.local.")
			Expect(routeEmitter.Flush()).ToNot(HaveOccurred())

			client := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
			Eventually(func() string {
				url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
				resp, err := client.Get(url)
				if err != nil {
					return "server is not listening yet"
				}
				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				return string(respBody)
			}).Should(MatchJSON(`{
				"env": "",
				"hosts": [
				{
					"ip_address": "192.168.0.2",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				}],
				"service": ""
			}`))

			sessionContents := string(session.Out.Contents())
			Expect(sessionContents).To(MatchRegexp(`HTTPServer access.*\"ip_address\\":\\"192.168.0.2\\".*\"serviceKey\":\"app-id.internal.local.\"`))
		})

		Context("when the route emitter greets the service discovery controller", func() {
			var greetMessageReply string
			BeforeEach(func() {
				greetMessageReply = "route-emitter-reply"
			})
			It("tells the route-emitter the minimum register interval as resume_pruning_delay_seconds", func(done Done) {
				routeEmitter.Subscribe(greetMessageReply, nats.MsgHandler(func(greetMsg *nats.Msg) {
					defer GinkgoRecover()
					msg := *greetMsg
					Expect(msg.Subject).To(Equal(greetMessageReply))
					subscriberOpts := struct {
						ID                               string
						MinimumRegisterIntervalInSeconds int
						PruneThresholdInSeconds          int
						AcceptTLS                        bool
					}{}
					Expect(json.Unmarshal(msg.Data, &subscriberOpts)).To(Succeed())
					Expect(subscriberOpts.MinimumRegisterIntervalInSeconds).To(Equal(1))
					Expect(subscriberOpts.PruneThresholdInSeconds).To(Equal(120))
					close(done)
				}))
				Expect(routeEmitter.PublishMsg(&nats.Msg{
					Subject: "service-discovery.greet",
					Reply:   greetMessageReply,
					Data:    []byte{},
				})).NotTo(HaveOccurred())
			}, 5 /* <-- overall spec timeout in seconds */)
		})

		It("should return a http app json", func() {
			url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
			resp, err := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert).Get(url)
			Expect(err).ToNot(HaveOccurred())
			respBody, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(respBody).To(MatchJSON(`{
				"env": "",
				"hosts": [
				{
					"ip_address": "192.168.0.1",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.2",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				}],
				"service": ""
			}`))
		})

		It("should return a http large json", func() {
			url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/large-id.internal.local.", port)
			resp, err := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert).Get(url)
			Expect(err).ToNot(HaveOccurred())
			respBody, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(respBody).To(MatchJSON(`{
				"env": "",
				"hosts": [
				{
					"ip_address": "192.168.0.1",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.2",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.3",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.4",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.5",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.6",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.7",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.8",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.9",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.10",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.11",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
				{
					"ip_address": "192.168.0.12",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				},
							{
					"ip_address": "192.168.0.13",
					"last_check_in": "",
					"port": 0,
					"revision": "",
					"service": "",
					"service_repo_name": "",
					"tags": {}
				}
				],
				"service": ""
			}`))
		})

		It("eventually removes stale routes", func() {
			client := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
			waitDuration := time.Duration(stalenessThresholdSeconds+pruningIntervalSeconds+1) * time.Second

			Eventually(func() []byte {
				url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
				resp, err := client.Get(url)
				Expect(err).ToNot(HaveOccurred())

				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				return respBody
			}, waitDuration).Should(MatchJSON(`{ "env": "", "hosts": [], "service": "" }`))
		})

		Context("when we hit the /routes endpoint", func() {
			It("should return a map of all hostnames to ips", func() {
				url := fmt.Sprintf("https://127.0.0.1:%d/routes", port)
				resp, err := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert).Get(url)
				Expect(err).ToNot(HaveOccurred())
				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				Expect(respBody).To(Or(MatchJSON(`{
					"addresses": [{
						"hostname": "app-id.internal.local.",
						"ips": [
							"192.168.0.1",
							"192.168.0.2"
							]
					}, {
						"hostname": "large-id.internal.local.",
						"ips": [
							"192.168.0.1",
							"192.168.0.2",
							"192.168.0.3",
							"192.168.0.4",
							"192.168.0.5",
							"192.168.0.6",
							"192.168.0.7",
							"192.168.0.8",
							"192.168.0.9",
							"192.168.0.10",
							"192.168.0.11",
							"192.168.0.12",
							"192.168.0.13"
							]
						}
					]
				}`),
					MatchJSON(`{
					"addresses": [
						{
							"hostname": "large-id.internal.local.",
							"ips": [
								"192.168.0.1",
								"192.168.0.2",
								"192.168.0.3",
								"192.168.0.4",
								"192.168.0.5",
								"192.168.0.6",
								"192.168.0.7",
								"192.168.0.8",
								"192.168.0.9",
								"192.168.0.10",
								"192.168.0.11",
								"192.168.0.12",
								"192.168.0.13"
							]
						}, {
							"hostname": "app-id.internal.local.",
							"ips": [
								"192.168.0.1",
								"192.168.0.2"
							]
						}
					]
				}`),
				))
			})
		})

		Context("when one of the nats urls is invalid", func() {
			BeforeEach(func() {
				os.Remove(configPath)
				garbagePort := ports.PickAPort()
				configPath = writeConfigFile(fmt.Sprintf(`{
					"address":"127.0.0.1",
					"port":"%d",
					"ca_cert": "%s",
					"server_cert": "%s",
					"server_key": "%s",
					"nats":[
						{
							"host":"garbage",
							"port":%d,
							"user":"who",
							"pass":"what"
						},
						{
							"host":"localhost",
							"port":%d,
							"user":"",
							"pass":""
						}
					],
					"staleness_threshold_seconds": %d,
					"pruning_interval_seconds": %d,
					"metrics_emit_seconds": 2,
					"metron_port": %d,
					"log_level_address": "%s",
				    "log_level_port": %d,
					"resume_pruning_delay_seconds": 0,
					"warm_duration_seconds": 0
				}`, port, caFile, serverCert, serverKey, garbagePort, natsServerPort, stalenessThresholdSeconds, pruningIntervalSeconds, fakeMetron.Port(), logLevelEndpointAddress, logLevelEndpointPort))
			})

			It("connects to NATs successfully", func() {
				Eventually(func() string {
					url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
					resp, err := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert).Get(url)
					Expect(err).ToNot(HaveOccurred())
					respBody, err := ioutil.ReadAll(resp.Body)
					Expect(err).ToNot(HaveOccurred())
					return string(respBody)
				}).Should(MatchJSON(`{
					"env": "",
					"hosts": [
					{
						"ip_address": "192.168.0.1",
						"last_check_in": "",
						"port": 0,
						"revision": "",
						"service": "",
						"service_repo_name": "",
						"tags": {}
					},
					{
						"ip_address": "192.168.0.2",
						"last_check_in": "",
						"port": 0,
						"revision": "",
						"service": "",
						"service_repo_name": "",
						"tags": {}
					}],
					"service": ""
				}`))
			})
		})

		Describe("emitting metrics", func() {
			withName := func(name string) types.GomegaMatcher {
				return WithTransform(func(ev metrics.Event) string {
					return ev.Name
				}, Equal(name))
			}
			withOrigin := func(origin string) types.GomegaMatcher {
				return WithTransform(func(ev metrics.Event) string {
					return ev.Origin
				}, Equal(origin))
			}

			It("emits an uptime metric", func() {
				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
					withName("uptime"),
					withOrigin("service-discovery-controller"),
				)))
			})

			Context("when a request is made to SDC", func() {
				JustBeforeEach(func() {
					url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
					_, err := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert).Get(url)
					Expect(err).ToNot(HaveOccurred())
				})

				It("emits a dns request metric", func() {
					Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
						withName("dnsRequest"),
						withOrigin("service-discovery-controller"),
					)))
				})

				It("emits a lookup duration", func() {
					Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
						withName("addressTableLookupTime"),
						withOrigin("service-discovery-controller"),
					)))
				})

			})
		})

		Context("when disconnected from the nats server", func() {
			var (
				client       *http.Client
				waitDuration time.Duration
			)

			BeforeEach(func() {
				client = testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
				waitDuration = time.Duration(stalenessThresholdSeconds+pruningIntervalSeconds+1) * time.Second
			})

			It("does not prune stale entries", func() {
				By("stopping the nats server and still returning routes past staleness threshold", func() {
					natsServer.Shutdown()
					Consistently(func() []byte {
						url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
						resp, err := client.Get(url)
						Expect(err).ToNot(HaveOccurred())

						respBody, err := ioutil.ReadAll(resp.Body)
						Expect(err).ToNot(HaveOccurred())

						return respBody
					}, waitDuration).Should(MatchJSON(`{
					"env": "",
					"hosts": [
					{
						"ip_address": "192.168.0.1",
						"last_check_in": "",
						"port": 0,
						"revision": "",
						"service": "",
						"service_repo_name": "",
						"tags": {}
					},
					{
						"ip_address": "192.168.0.2",
						"last_check_in": "",
						"port": 0,
						"revision": "",
						"service": "",
						"service_repo_name": "",
						"tags": {}
					}],
					"service": ""
				}`))
				})

				By("resuming pruning when nats server is back up", func() {
					natsServer = RunNatsServerOnPort(natsServerPort)
					Eventually(func() []byte {
						url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
						resp, err := client.Get(url)
						Expect(err).ToNot(HaveOccurred())

						respBody, err := ioutil.ReadAll(resp.Body)
						Expect(err).ToNot(HaveOccurred())

						return respBody
					}, waitDuration).Should(MatchJSON(`{
					"env": "",
					"hosts": [],
					"service": ""
				}`))
				})

			})
		})

		Context("Attempting to adjust log level", func() {
			It("it accepts the debug request", func() {
				response := requestLogChange("debug", logLevelEndpointPort)
				Expect(response.StatusCode).To(Equal(http.StatusNoContent))
				Eventually(session).Should(gbytes.Say("Log level set to DEBUG"))
			})

			It("it accepts the info request", func() {
				response := requestLogChange("info", logLevelEndpointPort)
				Expect(response.StatusCode).To(Equal(http.StatusNoContent))
				Eventually(session).Should(gbytes.Say("Log level set to INFO"))
			})

			It("it refuses the error request", func() {
				response := requestLogChange("error", logLevelEndpointPort)
				Expect(response.StatusCode).To(Equal(http.StatusBadRequest))
				Eventually(session).Should(gbytes.Say("Invalid log level requested: `error`. Skipping."))
			})

			It("it refuses the critical request", func() {
				response := requestLogChange("fatal", logLevelEndpointPort)
				Expect(response.StatusCode).To(Equal(http.StatusBadRequest))
				Eventually(session).Should(gbytes.Say("Invalid log level requested: `fatal`. Skipping."))
			})

			It("logs at info level by default", func() {
				client := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
				url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
				_, err := client.Get(url)
				Expect(err).ToNot(HaveOccurred())

				Expect(session).ToNot(gbytes.Say("HTTPServer access"))
			})

			It("logs at debug level when configured", func() {
				requestLogChange("debug", logLevelEndpointPort)
				client := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
				url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
				_, err := client.Get(url)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session).Should(gbytes.Say("HTTPServer access"))
			})

			It("logs at info level when switched back to info", func() {
				requestLogChange("debug", logLevelEndpointPort)
				requestLogChange("info", logLevelEndpointPort)

				client := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
				url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
				_, err := client.Get(url)
				Expect(err).ToNot(HaveOccurred())

				Expect(session).ToNot(gbytes.Say("HTTPServer access"))
			})
		})

		Context("when making requests before the warm duration is over", func() {
			BeforeEach(func() {
				os.Remove(configPath)
				configPath = writeConfigFile(fmt.Sprintf(`{
					"address":"127.0.0.1",
					"port":"%d",
					"ca_cert": "%s",
					"server_cert": "%s",
					"server_key": "%s",
					"nats":[
						{
							"host":"localhost",
							"port":%d,
							"user":"",
							"pass":""
						}
					],
					"staleness_threshold_seconds": %d,
					"pruning_interval_seconds": %d,
					"log_level_address": "%s",
					"log_level_port": %d,
					"metron_port": %d,
					"metrics_emit_seconds": 2,
					"resume_pruning_delay_seconds": 0,
					"warm_duration_seconds": 1
				}`,
					port, caFile, serverCert, serverKey, natsServerPort, stalenessThresholdSeconds, pruningIntervalSeconds, logLevelEndpointAddress, logLevelEndpointPort, fakeMetron.Port()))
			})

			It("returns 500 for requests before the warm duration is over", func() {
				url := fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port)
				client := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
				resp, err := client.Get(url)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

				Eventually(func() int {
					resp, err := client.Get(url)
					Expect(err).ToNot(HaveOccurred())
					return resp.StatusCode
				}).Should(Equal(http.StatusOK))
			})
		})
	})

	Context("when the log level endpoint fails to start successfully", func() {
		var conflictingServer *http.Server

		BeforeEach(func() {
			conflictingServer = testhelpers.LaunchConflictingServer(logLevelEndpointPort)
		})

		AfterEach(func() {
			conflictingServer.Close()
		})

		It("exits", func() {
			startCmd := exec.Command(pathToServer, "-c", configPath)
			var err error
			session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session).Should(gexec.Exit(2))
			Eventually(session, 5*time.Second).Should(gbytes.Say("service-discovery-controller.log-level-server.Listen and serve exited with error:"))
		})
	})

	Context("when none of the nats urls are valid", func() {
		BeforeEach(func() {
			os.Remove(configPath)
			garbagePort := ports.PickAPort()
			configPath = writeConfigFile(fmt.Sprintf(`{
				"port":"%d",
				"address":"127.0.0.1",
				"ca_cert": "%s",
				"server_cert": "%s",
				"server_key": "%s",
				"nats":[
					{
						"host":"garbage",
						"port":%d,
						"user":"who",
						"pass":"what"
					}
				],
				"staleness_threshold_seconds": %d,
				"pruning_interval_seconds": %d,
				"log_level_address": "%s",
				"log_level_port": %d,
				"metron_port": %d,
				"metrics_emit_seconds": 2,
				"resume_pruning_delay_seconds": 0,
				"warm_duration_seconds": 0
			}`, port, caFile, serverCert, serverKey, garbagePort, stalenessThresholdSeconds, pruningIntervalSeconds, logLevelEndpointAddress, logLevelEndpointPort, fakeMetron.Port()))
		})

		It("fails to start successfully", func() {
			startCmd := exec.Command(pathToServer, "-c", configPath)
			var err error
			session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session, 5*time.Second).Should(gexec.Exit(2))
			Expect(session).To(gbytes.Say("service-discovery-controller.*unable to create nats connection: nats: no servers available for connection"))
		})
	})
})

func requestLogChange(logLevel string, port int) *http.Response {
	client := &http.Client{}
	postBody := strings.NewReader(logLevel)
	url := fmt.Sprintf("http://localhost:%d/log-level", port)
	response, err := client.Post(url, "text/plain", postBody)
	Expect(err).ToNot(HaveOccurred())
	return response
}

func register(routeEmitter *nats.Conn, ip string, url string) {
	natsRegistryMsg := nats.Msg{
		Subject: "service-discovery.register",
		Data:    []byte(fmt.Sprintf(`{"host": "%s","uris":["%s"]}`, ip, url)),
	}

	Expect(routeEmitter.PublishMsg(&natsRegistryMsg)).ToNot(HaveOccurred())
}

func unregister(routeEmitter *nats.Conn, ip string, url string) {
	natsRegistryMsg := nats.Msg{
		Subject: "service-discovery.unregister",
		Data:    []byte(fmt.Sprintf(`{"host": "%s","uris":["%s"]}`, ip, url)),
	}

	Expect(routeEmitter.PublishMsg(&natsRegistryMsg)).ToNot(HaveOccurred())
}

func newFakeRouteEmitter(natsUrl string) *nats.Conn {
	natsClient, err := nats.Connect(natsUrl, nats.ReconnectWait(1*time.Nanosecond))
	Expect(err).NotTo(HaveOccurred())
	return natsClient
}

func writeConfigFile(configJson string) string {
	configFile, err := ioutil.TempFile(os.TempDir(), "sd_config")
	Expect(err).ToNot(HaveOccurred())

	configPath := configFile.Name()

	err = ioutil.WriteFile(configPath, []byte(configJson), os.ModePerm)
	Expect(err).ToNot(HaveOccurred())

	return configPath
}
