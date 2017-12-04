package nonmutualtls_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"lib/nonmutualtls"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

const START_TIMEOUT = 10 * time.Second
const WAIT_TIMEOUT = 2 * time.Second

var _ = Describe("TLS config for internal API server", func() {
	var (
		serverListenAddr string
		clientTLSConfig  *tls.Config
		serverTLSConfig  *tls.Config
	)

	BeforeEach(func() {
		var err error

		port := ports.PickAPort()
		serverListenAddr = fmt.Sprintf("127.0.0.1:%d", port)
		clientTLSConfig, err = nonmutualtls.NewClientTLSConfig(paths.ServerCACertPath)
		Expect(err).NotTo(HaveOccurred())
		serverTLSConfig, err = nonmutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath)
		Expect(err).NotTo(HaveOccurred())
	})

	startServer := func(tlsConfig *tls.Config) ifrit.Process {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		})
		someServer := http_server.NewTLSServer(serverListenAddr, testHandler, tlsConfig)

		members := grouper.Members{{
			Name:   "http_server",
			Runner: someServer,
		}}
		group := grouper.NewOrdered(os.Interrupt, members)
		monitor := ifrit.Invoke(sigmon.New(group))

		Expect(testsupport.WaitOrReady(START_TIMEOUT, monitor)).To(Succeed())
		return monitor
	}

	makeRequest := func(serverAddr string, clientTLSConfig *tls.Config) (*http.Response, error) {
		req, err := http.NewRequest("GET", "https://"+serverAddr+"/", nil)
		Expect(err).NotTo(HaveOccurred())
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: clientTLSConfig,
			},
		}
		return client.Do(req)
	}

	Describe("Server TLS Config", func() {
		It("returns a TLSConfig that can be used by an HTTP server", func() {
			server := startServer(serverTLSConfig)

			resp, err := makeRequest(serverListenAddr, clientTLSConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			respBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(respBytes).To(Equal([]byte("hello")))
			Expect(resp.Body.Close()).To(Succeed())

			server.Signal(os.Interrupt)
			Eventually(server.Wait(), WAIT_TIMEOUT).Should(Receive())
		})

		Context("when the key pair cannot be created", func() {
			It("returns a meaningful error", func() {
				_, err := nonmutualtls.NewServerTLSConfig("", "")
				Expect(err).To(MatchError(HavePrefix("unable to load cert or key")))
			})
		})

		Context("when it is misconfigured", func() {
			var server ifrit.Process
			BeforeEach(func() {
				server = startServer(serverTLSConfig)
			})

			AfterEach(func() {
				server.Signal(os.Interrupt)
				Eventually(server.Wait(), WAIT_TIMEOUT).Should(Receive())
			})

			Context("when the client has been configured without a CA", func() {
				BeforeEach(func() {
					clientTLSConfig.RootCAs = nil
				})

				It("refuses to connect to the server", func() {
					_, err := makeRequest(serverListenAddr, clientTLSConfig)
					Expect(err).To(MatchError(ContainSubstring("x509: certificate signed by unknown authority")))
				})
			})

			Context("when the client has been configured with the wrong CA for the server", func() {
				BeforeEach(func() {
					wrongServerCACert, err := ioutil.ReadFile(paths.WrongServerCACertPath)
					Expect(err).NotTo(HaveOccurred())

					clientCertPool := x509.NewCertPool()
					clientCertPool.AppendCertsFromPEM(wrongServerCACert)
					clientTLSConfig.RootCAs = clientCertPool
				})

				It("refuses to connect to the server", func() {
					_, err := makeRequest(serverListenAddr, clientTLSConfig)
					Expect(err).To(MatchError(ContainSubstring("x509: certificate signed by unknown authority")))
				})
			})

			Context("when the client is configured to use an unsupported ciphersuite", func() {
				BeforeEach(func() {
					clientTLSConfig.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_256_GCM_SHA384}
				})

				It("refuses the connection from the client", func() {
					_, err := makeRequest(serverListenAddr, clientTLSConfig)
					Expect(err).To(MatchError(ContainSubstring("remote error")))
				})
			})

			Context("when the client is configured to use TLS 1.1", func() {
				BeforeEach(func() {
					clientTLSConfig.MinVersion = tls.VersionTLS11
					clientTLSConfig.MaxVersion = tls.VersionTLS11
				})

				It("refuses the connection from the client", func() {
					_, err := makeRequest(serverListenAddr, clientTLSConfig)
					Expect(err).To(MatchError(ContainSubstring("remote error")))
				})
			})
		})
	})
})
