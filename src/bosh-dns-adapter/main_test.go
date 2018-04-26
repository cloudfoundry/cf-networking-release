package main_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"test-helpers"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Main", func() {
	var (
		session                                *gexec.Session
		tempConfigFile                         *os.File
		configFileContents                     string
		fakeServiceDiscoveryControllerServer   *ghttp.Server
		fakeServiceDiscoveryControllerResponse []http.HandlerFunc
		dnsAdapterAddress                      string
		dnsAdapterPort                         string
		fakeMetron                             metrics.FakeMetron
		logLevelPort                           int
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		fakeServiceDiscoveryControllerResponse = []http.HandlerFunc{ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/v1/registration/app-id.internal.local."),
			ghttp.RespondWith(200, `{
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
					}],
					"service": ""
				}`),
		)}
		dnsAdapterAddress = "127.0.0.1"

		dnsAdapterPort = fmt.Sprintf("%d", ports.PickAPort())
		logLevelPort = ports.PickAPort()
	})

	JustBeforeEach(func() {
		var err error
		caFileName, clientCertFileName, clientKeyFileName, serverCert := testhelpers.GenerateCaAndMutualTlsCerts()

		fakeServiceDiscoveryControllerServer = ghttp.NewUnstartedServer()
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS = &tls.Config{}
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.RootCAs = testhelpers.CertPool(caFileName)
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.ClientCAs = testhelpers.CertPool(caFileName)
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.ClientAuth = tls.RequireAndVerifyClientCert
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.PreferServerCipherSuites = true
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.Certificates = []tls.Certificate{serverCert}

		fakeServiceDiscoveryControllerServer.AppendHandlers(fakeServiceDiscoveryControllerResponse...)
		fakeServiceDiscoveryControllerServer.HTTPTestServer.StartTLS()

		urlParts := strings.Split(fakeServiceDiscoveryControllerServer.URL(), ":")

		configFileContents = fmt.Sprintf(`{
			"address": "%s",
			"port": "%s",
			"service_discovery_controller_address": "%s",
			"service_discovery_controller_port": "%s",
			"client_cert": "%s",
			"client_key": "%s",
			"ca_cert": "%s",
			"metron_port": %d,
			"metrics_emit_seconds": 2,
			"log_level_port": %d,
			"log_level_address": "127.0.0.1"
		}`, dnsAdapterAddress,
			dnsAdapterPort,
			strings.TrimPrefix(urlParts[1], "//"),
			urlParts[2],
			clientCertFileName,
			clientKeyFileName,
			caFileName,
			fakeMetron.Port(),
			logLevelPort,
		)

		tempConfigFile, err = ioutil.TempFile(os.TempDir(), "sd")
		Expect(err).ToNot(HaveOccurred())
		_, err = tempConfigFile.Write([]byte(configFileContents))
		Expect(err).ToNot(HaveOccurred())

		startCmd := exec.Command(pathToServer, "-c", tempConfigFile.Name())
		session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		session.Kill()
		os.Remove(tempConfigFile.Name())

		fakeServiceDiscoveryControllerServer.Close()
	})

	It("should return a http 200 status", func() {
		Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))

		var reader io.Reader
		url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
		request, err := http.NewRequest("GET", url, reader)
		Expect(err).To(Succeed())

		resp, err := http.DefaultClient.Do(request)
		Expect(err).To(Succeed())

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		all, err := ioutil.ReadAll(resp.Body)
		Expect(err).To(Succeed())
		Expect(string(all)).To(MatchJSON(`{
					"Status": 0,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1
						}
					],
					"Answer":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}
		`))
	})

	It("accepts interrupt signals and shuts down", func() {
		Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))
		session.Signal(os.Interrupt)

		Eventually(session).Should(gexec.Exit())
		Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-stopped"))
	})

	Describe("emitting metrics", func() {
		Context("when things are going well", func() {
			JustBeforeEach(func() {
				Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))

				url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
				makeDNSRequest(url, http.StatusOK)
			})

			It("emits an uptime metric", func() {
				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
					metricWithName("uptime"),
					metricWithOrigin("bosh-dns-adapter"),
				)))
			})

			It("emits an request metrics", func() {
				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
					metricWithName("GetIPsRequestTime"),
					metricWithOrigin("bosh-dns-adapter"),
				)))

				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
					metricWithName("GetIPsRequestCount"),
					metricWithOrigin("bosh-dns-adapter"),
				)))
			})
		})

		Context("when things don't go well", func() {
			JustBeforeEach(func() {
				Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))

				fakeServiceDiscoveryControllerServer.Close()

				url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
				makeDNSRequest(url, http.StatusInternalServerError)
			})

			It("emits failed request metrics", func() {
				Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
					metricWithName("DNSRequestFailures"),
					metricWithOrigin("bosh-dns-adapter"),
				)))
			})
		})
	})

	Context("when a process is already listening on the port", func() {
		var session2 *gexec.Session
		JustBeforeEach(func() {
			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))
			startCmd := exec.Command(pathToServer, "-c", tempConfigFile.Name())
			var err error
			session2, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session2.Kill().Wait()
		})

		It("fails to start", func() {
			Eventually(session2, 5*time.Second).Should(gexec.Exit(1))
			expectedErrStr := fmt.Sprintf("Address \\(127.0.0.1:%s\\) not available", dnsAdapterPort)
			Eventually(session2.Err).Should(gbytes.Say(expectedErrStr))
		})
	})

	Context("when 'type' url param is not provided", func() {
		It("should default to type A record", func() {
			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))

			var reader io.Reader
			url := fmt.Sprintf("http://127.0.0.1:%s?name=app-id.internal.local.", dnsAdapterPort)
			request, err := http.NewRequest("GET", url, reader)
			Expect(err).To(Succeed())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

			all, err := ioutil.ReadAll(resp.Body)
			Expect(err).To(Succeed())
			Expect(string(all)).To(MatchJSON(`{
					"Status": 0,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1
						}
					],
					"Answer":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}
		`))
		})
	})

	Context("when 'name' url param is not provided", func() {
		It("returns a http 400 status", func() {
			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))
			var reader io.Reader
			url := fmt.Sprintf("http://127.0.0.1:%s?type=1", dnsAdapterPort)
			request, err := http.NewRequest("GET", url, reader)
			Expect(err).To(Succeed())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			all, err := ioutil.ReadAll(resp.Body)
			Expect(err).To(Succeed())

			Expect(string(all)).To(MatchJSON(`{
					"Status": 2,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "",
							"type": 1
						}
					],
					"Answer": [ ],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when configured with an invalid port", func() {
		BeforeEach(func() {
			dnsAdapterPort = "-1"
		})

		It("should fail to startup", func() {
			Eventually(session).Should(gexec.Exit(1))
		})
	})

	Context("when configured with an invalid config file path", func() {
		var session2 *gexec.Session
		JustBeforeEach(func() {
			startCmd := exec.Command(pathToServer, "-c", "/non-existent-path")
			var err error
			session2, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session2.Kill().Wait()
		})

		It("should fail to startup", func() {
			Eventually(session2).Should(gexec.Exit(2))
			Eventually(session2).Should(gbytes.Say("Could not read config file"))
			Eventually(session2).Should(gbytes.Say("/non-existent-path"))
		})
	})

	Context("when configured garbage config file content", func() {
		BeforeEach(func() {
			dnsAdapterAddress = `"garbage`
		})

		It("should fail to startup", func() {
			Eventually(session).Should(gexec.Exit(2))
			Eventually(session).Should(gbytes.Say("Could not parse config file"))
			Eventually(session).Should(gbytes.Say(tempConfigFile.Name()))
		})
	})

	Context("when no config file is passed", func() {
		var session2 *gexec.Session
		JustBeforeEach(func() {
			startCmd := exec.Command(pathToServer)
			var err error
			session2, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session2.Kill().Wait()
		})

		It("should fail to startup", func() {
			Eventually(session2).Should(gexec.Exit(2))
		})
	})

	Context("when requesting anything but an A record", func() {
		It("should return a successful response with no answers", func() {
			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))
			url := fmt.Sprintf("http://127.0.0.1:%s?type=16&name=app-id.internal.local.", dnsAdapterPort)
			request, err := http.NewRequest("GET", url, nil)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			all, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(all)).To(MatchJSON(`{
					"Status": 0,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app-id.internal.local.",
							"type": 16
						}
					],
					"Answer": [ ],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when the service discovery controller returns non-successful", func() {
		BeforeEach(func() {
			fakeServiceDiscoveryControllerResponse = []http.HandlerFunc{
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/registration/app-id.internal.local."),
					ghttp.RespondWith(404, `{ }`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/registration/app-id.internal.local."),
					ghttp.RespondWith(404, `{ }`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/registration/app-id.internal.local."),
					ghttp.RespondWith(404, `{ }`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/registration/app-id.internal.local."),
					ghttp.RespondWith(404, `{ }`),
				),
			}
		})

		It("returns a 500 and an error", func() {
			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))
			var reader io.Reader

			url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
			request, err := http.NewRequest("GET", url, reader)
			Expect(err).To(Succeed())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
				metricWithName("DNSRequestFailures"),
				metricWithOrigin("bosh-dns-adapter"),
			)))
		})
	})

	Context("logging", func() {
		JustBeforeEach(func() {
			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))
			response := requestLogChange("debug", logLevelPort)
			Expect(response.StatusCode).To(Equal(http.StatusNoContent))
		})

		Context("When making a request for a hostname with an associated ip", func() {
			JustBeforeEach(func() {
				url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
				makeDNSRequest(url, 200)
			})

			It("logs the request with app domain and ips", func() {
				Eventually(session).Should(gbytes.Say("bosh-dns-adapter.serve-request.*192.168.0.1.*app-id.internal.local"))
			})
		})

		Context("When making a request for a non A record", func() {
			JustBeforeEach(func() {
				url := fmt.Sprintf("http://127.0.0.1:%s?type=2&name=app-id.internal.local.", dnsAdapterPort)
				makeDNSRequest(url, 200)
			})

			It("logs the request with app domain and notifies of the un-supported record type", func() {
				Eventually(session).Should(gbytes.Say("bosh-dns-adapter.serve-request.*unsupported record type.*app-id.internal.local"))
			})
		})

		Context("When making a request without a domain name", func() {
			JustBeforeEach(func() {
				url := fmt.Sprintf("http://127.0.0.1:%s?type=1", dnsAdapterPort)
				makeDNSRequest(url, 400)
			})

			It("logs the request and notifies of the missing name", func() {
				Eventually(session).Should(gbytes.Say("bosh-dns-adapter.serve-request.*name parameter empty"))
			})
		})

		Context("When making a request to the sdc fails", func() {
			JustBeforeEach(func() {
				fakeServiceDiscoveryControllerServer.Close()
				url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
				makeDNSRequest(url, 500)
			})

			It("logs the error", func() {
				Eventually(session).Should(gbytes.Say("bosh-dns-adapter.serve-request.*could not connect to service discovery controller"))
			})
		})
	})

	Context("Attempting to adjust log level", func() {
		JustBeforeEach(func() {
			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.server-started"))
		})

		It("it accepts the debug request", func() {
			response := requestLogChange("debug", logLevelPort)
			Expect(response.StatusCode).To(Equal(http.StatusNoContent))
			Eventually(session).Should(gbytes.Say("Log level set to DEBUG"))
		})

		It("it accepts the info request", func() {
			response := requestLogChange("info", logLevelPort)
			Expect(response.StatusCode).To(Equal(http.StatusNoContent))
			Eventually(session).Should(gbytes.Say("Log level set to INFO"))
		})

		It("it refuses the error request", func() {
			response := requestLogChange("error", logLevelPort)
			Expect(response.StatusCode).To(Equal(http.StatusBadRequest))
			Eventually(session).Should(gbytes.Say("Invalid log level requested: `error`. Skipping."))
		})

		It("it refuses the critical request", func() {
			response := requestLogChange("fatal", logLevelPort)
			Expect(response.StatusCode).To(Equal(http.StatusBadRequest))
			Eventually(session).Should(gbytes.Say("Invalid log level requested: `fatal`. Skipping."))
		})

		It("logs at info level by default", func() {
			url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
			request, err := http.NewRequest("GET", url, nil)
			resp, err := http.DefaultClient.Do(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			Expect(session).ToNot(gbytes.Say("bosh-dns-adapter.serve-request"))
		})

		It("logs at debug level when configured", func() {
			requestLogChange("debug", logLevelPort)

			url := fmt.Sprintf("http://127.0.0.1:%s?type=1&name=app-id.internal.local.", dnsAdapterPort)
			request, err := http.NewRequest("GET", url, nil)
			resp, err := http.DefaultClient.Do(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			Eventually(session).Should(gbytes.Say("bosh-dns-adapter.serve-request"))
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

func makeDNSRequest(url string, expectedResponseCode int) {
	request, err := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(request)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(expectedResponseCode))
}

func metricWithName(name string) types.GomegaMatcher {
	return WithTransform(func(ev metrics.Event) string {
		return ev.Name
	}, Equal(name))
}

func metricWithOrigin(origin string) types.GomegaMatcher {
	return WithTransform(func(ev metrics.Event) string {
		return ev.Origin
	}, Equal(origin))
}
