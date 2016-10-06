package mutualtls_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"policy-server/mutualtls"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var _ = Describe("TLS config for internal API server", func() {
	var (
		serverListenAddr      string
		clientTLSConfig       *tls.Config
		clientCACert          []byte
		serverCert, serverKey []byte
	)

	BeforeEach(func() {
		var err error
		serverCert, err = ioutil.ReadFile("../integration/fixtures/server.crt")
		Expect(err).NotTo(HaveOccurred())
		serverKey, err = ioutil.ReadFile("../integration/fixtures/server.key")
		Expect(err).NotTo(HaveOccurred())

		serverListenAddr = fmt.Sprintf("127.0.0.1:%d", 40000+rand.Intn(10000))
		cert, err := tls.LoadX509KeyPair("../integration/fixtures/client.crt", "../integration/fixtures/client.key")
		Expect(err).NotTo(HaveOccurred())

		clientCACert, err = ioutil.ReadFile("../integration/fixtures/netman-ca.crt")
		Expect(err).NotTo(HaveOccurred())

		clientCertPool := x509.NewCertPool()
		clientCertPool.AppendCertsFromPEM(clientCACert)

		clientTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      clientCertPool,
		}
		clientTLSConfig.BuildNameToCertificate()
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

		Eventually(monitor.Ready()).Should(BeClosed())
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

	Describe("BuildConfig", func() {
		It("returns a TLSConfig that can be used by an HTTP server", func() {
			serverTLSConfig, err := mutualtls.BuildConfig(serverCert, serverKey, clientCACert)
			Expect(err).NotTo(HaveOccurred())

			server := startServer(serverTLSConfig)

			resp, err := makeRequest(serverListenAddr, clientTLSConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			respBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(respBytes).To(Equal([]byte("hello")))
			Expect(resp.Body.Close()).To(Succeed())

			server.Signal(os.Interrupt)
			Eventually(server.Wait()).Should(Receive())
		})

		Context("when the key pair cannot be created", func() {
			It("returns a meaningful error", func() {
				_, err := mutualtls.BuildConfig(nil, nil, nil)
				Expect(err).To(MatchError(HavePrefix("unable to load server cert or key")))
			})
		})

		Context("when the server has been configured with the wrong CA for the client", func() {
			BeforeEach(func() {
				var err error
				clientCACert, err = ioutil.ReadFile("../integration/fixtures/wrong-netman-ca.crt")
				Expect(err).NotTo(HaveOccurred())
			})

			It("refuses to connect to the client", func() {
				serverTLSConfig, err := mutualtls.BuildConfig(serverCert, serverKey, clientCACert)
				Expect(err).NotTo(HaveOccurred())

				server := startServer(serverTLSConfig)

				_, err = makeRequest(serverListenAddr, clientTLSConfig)
				Expect(err).To(MatchError(ContainSubstring("remote error")))

				server.Signal(os.Interrupt)
				Eventually(server.Wait()).Should(Receive())
			})
		})

		Context("when the client has been configured without a CA", func() {
			BeforeEach(func() {
				clientTLSConfig.RootCAs = nil
			})

			It("refuses to connect to the server", func() {
				serverTLSConfig, err := mutualtls.BuildConfig(serverCert, serverKey, clientCACert)
				Expect(err).NotTo(HaveOccurred())

				server := startServer(serverTLSConfig)

				_, err = makeRequest(serverListenAddr, clientTLSConfig)
				Expect(err).To(MatchError(ContainSubstring("x509: certificate signed by unknown authority")))

				server.Signal(os.Interrupt)
				Eventually(server.Wait()).Should(Receive())
			})
		})

		Context("when the client does not present client certificates to the server", func() {
			BeforeEach(func() {
				clientTLSConfig.Certificates = nil
			})

			It("refuses the connection from the client", func() {
				serverTLSConfig, err := mutualtls.BuildConfig(serverCert, serverKey, clientCACert)
				Expect(err).NotTo(HaveOccurred())

				server := startServer(serverTLSConfig)

				_, err = makeRequest(serverListenAddr, clientTLSConfig)
				Expect(err).To(MatchError(ContainSubstring("remote error")))

				server.Signal(os.Interrupt)
				Eventually(server.Wait()).Should(Receive())
			})
		})

		Context("when the client presents certificates that the server does not trust", func() {
			BeforeEach(func() {
				invalidClient, err := tls.LoadX509KeyPair("../integration/fixtures/wrong-client.crt", "../integration/fixtures/wrong-client.key")
				Expect(err).NotTo(HaveOccurred())
				clientTLSConfig.Certificates = []tls.Certificate{invalidClient}
			})

			It("refuses the connection from the client", func() {
				serverTLSConfig, err := mutualtls.BuildConfig(serverCert, serverKey, clientCACert)
				Expect(err).NotTo(HaveOccurred())

				server := startServer(serverTLSConfig)

				_, err = makeRequest(serverListenAddr, clientTLSConfig)
				Expect(err).To(MatchError(ContainSubstring("remote error")))

				server.Signal(os.Interrupt)
				Eventually(server.Wait()).Should(Receive())
			})
		})

		It("returns config with reasonable security properties", func() {
			serverTLSConfig, err := mutualtls.BuildConfig(serverCert, serverKey, clientCACert)
			Expect(err).NotTo(HaveOccurred())

			Expect(serverTLSConfig.PreferServerCipherSuites).To(BeTrue())
			Expect(serverTLSConfig.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
		})
	})
})
