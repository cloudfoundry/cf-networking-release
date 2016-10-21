package mutualtls_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"lib/mutualtls"
	"math/rand"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var _ = Describe("TLS config for internal API server", func() {
	var (
		serverListenAddr string
		clientTLSConfig  *tls.Config
	)

	BeforeEach(func() {
		var err error
		serverListenAddr = fmt.Sprintf("127.0.0.1:%d", 40000+rand.Intn(10000))
		clientTLSConfig, err = mutualtls.NewClientTLSConfig(paths.ClientCertPath, paths.ClientKeyPath, paths.ServerCACertPath)
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

	Describe("NewServerTLSConfig", func() {
		It("returns a TLSConfig that can be used by an HTTP server", func() {
			serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath, paths.ClientCACertPath)
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
				_, err := mutualtls.NewServerTLSConfig("", "", "")
				Expect(err).To(MatchError(HavePrefix("unable to load cert or key")))
			})
		})

		Context("when the server has been configured with the wrong CA for the client", func() {
			It("refuses to connect to the client", func() {
				serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath, paths.WrongClientCACertPath)
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
				serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath, paths.ClientCACertPath)
				Expect(err).NotTo(HaveOccurred())

				server := startServer(serverTLSConfig)

				_, err = makeRequest(serverListenAddr, clientTLSConfig)
				Expect(err).To(MatchError(ContainSubstring("x509: certificate signed by unknown authority")))

				server.Signal(os.Interrupt)
				Eventually(server.Wait()).Should(Receive())
			})
		})

		Context("when the client has been configured with the wrong CA for the server", func() {
			BeforeEach(func() {
				wrongServerCACert, err := ioutil.ReadFile(paths.ClientCACertPath)
				Expect(err).NotTo(HaveOccurred())

				clientCertPool := x509.NewCertPool()
				clientCertPool.AppendCertsFromPEM(wrongServerCACert)
				clientTLSConfig.RootCAs = clientCertPool
			})

			It("refuses to connect to the server", func() {
				serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath, paths.ClientCACertPath)
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
				serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath, paths.ClientCACertPath)
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
				invalidClient, err := tls.LoadX509KeyPair(paths.WrongClientCertPath, paths.WrongClientKeyPath)
				Expect(err).NotTo(HaveOccurred())
				clientTLSConfig.Certificates = []tls.Certificate{invalidClient}
			})

			It("refuses the connection from the client", func() {
				serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath, paths.ClientCACertPath)
				Expect(err).NotTo(HaveOccurred())

				server := startServer(serverTLSConfig)

				_, err = makeRequest(serverListenAddr, clientTLSConfig)
				Expect(err).To(MatchError(ContainSubstring("remote error")))

				server.Signal(os.Interrupt)
				Eventually(server.Wait()).Should(Receive())
			})
		})

		It("returns config with reasonable security properties", func() {
			serverTLSConfig, err := mutualtls.NewServerTLSConfig(paths.ServerCertPath, paths.ServerKeyPath, paths.ClientCACertPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(serverTLSConfig.PreferServerCipherSuites).To(BeTrue())
			Expect(serverTLSConfig.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
			Expect(serverTLSConfig.CipherSuites).To(Equal([]uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}))
		})
	})
})
