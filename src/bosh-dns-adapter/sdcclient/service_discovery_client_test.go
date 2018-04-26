package sdcclient_test

import (
	"net/http"

	. "bosh-dns-adapter/sdcclient"
	"test-helpers"

	"crypto/tls"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("ServiceDiscoveryClient", func() {
	var (
		client             *ServiceDiscoveryClient
		fakeServer         *ghttp.Server
		fakeServerResponse http.HandlerFunc

		caFileName         string
		clientCertFileName string
		clientKeyFileName  string
		serverCert         tls.Certificate
	)

	BeforeEach(func() {
		caFileName, clientCertFileName, clientKeyFileName, serverCert = testhelpers.GenerateCaAndMutualTlsCerts()
	})

	Describe("NewServiceDiscoveryClient", func() {
		Context("when the client has a misconfigured CA path", func() {
			BeforeEach(func() {
				os.Remove(caFileName)
				caFileName = "non-existent"
			})

			It("returns an error", func() {
				_, err := NewServiceDiscoveryClient("app-id.apps.internal.", caFileName, clientCertFileName, clientKeyFileName)
				Expect(err).To(MatchError("read CA file: open non-existent: no such file or directory"))

			})
		})

		Context("when the client has a CA file that is malformed", func() {
			BeforeEach(func() {
				ioutil.WriteFile(caFileName, []byte("not a cert"), os.ModePerm)
			})

			It("returns an error", func() {
				_, err := NewServiceDiscoveryClient("app-id.apps.internal.", caFileName, clientCertFileName, clientKeyFileName)
				Expect(err).To(MatchError("load CA file into cert pool"))
			})
		})

		Context("when the client has a misconfigured client/key", func() {
			BeforeEach(func() {
				os.Remove(clientCertFileName)
				clientCertFileName = "non-existent"
			})
			It("returns an error", func() {
				_, err := NewServiceDiscoveryClient("app-id.apps.internal.", caFileName, clientCertFileName, clientKeyFileName)
				Expect(err).To(MatchError("load client key pair: open non-existent: no such file or directory"))
			})
		})

	})

	Describe("IPs", func() {
		BeforeEach(func() {
			fakeServer = ghttp.NewUnstartedServer()
			fakeServer.HTTPTestServer.TLS = &tls.Config{}
			fakeServer.HTTPTestServer.TLS.RootCAs = testhelpers.CertPool(caFileName)
			fakeServer.HTTPTestServer.TLS.ClientCAs = testhelpers.CertPool(caFileName)
			fakeServer.HTTPTestServer.TLS.ClientAuth = tls.RequireAndVerifyClientCert
			fakeServer.HTTPTestServer.TLS.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}
			fakeServer.HTTPTestServer.TLS.PreferServerCipherSuites = true
			fakeServer.HTTPTestServer.TLS.Certificates = []tls.Certificate{serverCert}
		})

		JustBeforeEach(func() {
			var err error
			fakeServer.HTTPTestServer.StartTLS()
			client, err = NewServiceDiscoveryClient(fakeServer.URL(), caFileName, clientCertFileName, clientKeyFileName)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			fakeServer.Close()
			os.Remove(caFileName)
			os.Remove(clientCertFileName)
			os.Remove(clientKeyFileName)
		})

		Context("when the server responds successfully", func() {
			BeforeEach(func() {
				fakeServerResponse = ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
					ghttp.RespondWith(http.StatusOK, `{
							"env": "",
							"Hosts": [
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
				fakeServer.AppendHandlers(fakeServerResponse)
			})

			It("returns the ips in the server response", func() {
				actualIPs, err := client.IPs("app-id.apps.internal.")
				Expect(err).ToNot(HaveOccurred())

				Expect(actualIPs).To(ConsistOf("192.168.0.1", "192.168.0.2"))
			})

		})

		Context("returned ips order", func() {
			BeforeEach(func() {
				fakeServer.RouteToHandler("GET", "/v1/registration/app-id.apps.internal.", func(writer http.ResponseWriter, request *http.Request) {
					_, err := writer.Write([]byte(`{
							"env": "",
							"Hosts": [
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
							}],
							"service": ""
						}`))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			It("shuffles them to return them in random order", func() {
				Eventually(func() []string {
					ips, err := client.IPs("app-id.apps.internal.")
					Expect(err).ToNot(HaveOccurred())
					return ips
				}).Should(Equal([]string{"192.168.0.3", "192.168.0.1", "192.168.0.2"}))
			})
		})

		Context("when the server responds with malformed JSON", func() {
			BeforeEach(func() {
				fakeServerResponse = ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
					ghttp.RespondWith(http.StatusOK, `garbage`))
				fakeServer.AppendHandlers(fakeServerResponse)
			})

			It("returns an error", func() {
				_, err := client.IPs("app-id.apps.internal.")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the server responds several non-200 responses, but eventually returns a 200 response", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusBadRequest, `{}`)),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusBadRequest, `{}`)),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusBadRequest, `{}`)),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusOK, `{
							"env": "",
							"Hosts": [
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
						}`)),
				)
			})

			It("retries and returns the successful response", func() {
				actualIPs, err := client.IPs("app-id.apps.internal.")
				Expect(err).ToNot(HaveOccurred())

				Expect(actualIPs).To(ConsistOf("192.168.0.1", "192.168.0.2"))
			})
		})

		Context("when the server responds several non-200 responses", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusBadRequest, `{}`)),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusBadRequest, `{}`)),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusBadRequest, `{}`)),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/v1/registration/app-id.apps.internal.", ""),
						ghttp.RespondWith(http.StatusBadRequest, `{}`)),
				)
			})

			It("returns an error", func() {
				_, err := client.IPs("app-id.apps.internal.")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Received non successful response from server:"))
			})
		})
	})

})
