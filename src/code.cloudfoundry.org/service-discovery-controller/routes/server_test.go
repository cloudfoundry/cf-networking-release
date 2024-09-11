package routes_test

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/service-discovery-controller/config"
	. "code.cloudfoundry.org/service-discovery-controller/routes"
	"code.cloudfoundry.org/service-discovery-controller/routes/fakes"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Server", func() {
	var (
		addressTable       *fakes.AddressTable
		dnsRequestRecorder *fakes.DNSRequestRecorder
		metricsSender      *fakes.MetricsSender
		clientCert         tls.Certificate
		caFile             string
		serverCert         string
		serverKey          string
		serverProc         ifrit.Process
		testLogger         *lagertest.TestLogger
		client             *http.Client
		tlsConfig          *tls.Config
		server             *Server
		port               int
	)

	BeforeEach(func() {
		caFile, serverCert, serverKey, clientCert = testhelpers.GenerateCaAndMutualTlsCerts()

		port = ports.PickAPort()

		testLogger = lagertest.NewTestLogger("test")
		config := &config.Config{
			Port:              strconv.Itoa(port),
			Address:           "127.0.0.1",
			CACert:            caFile,
			ServerCert:        serverCert,
			ServerKey:         serverKey,
			ReadHeaderTimeout: config.DurationFlag(200 * time.Millisecond),
		}
		addressTable = &fakes.AddressTable{}
		dnsRequestRecorder = &fakes.DNSRequestRecorder{}
		metricsSender = &fakes.MetricsSender{}
		server = NewServer(addressTable, config, dnsRequestRecorder, metricsSender, testLogger)
		certPool := testhelpers.CertPool(caFile)
		client = testhelpers.NewClient(certPool, clientCert)
		tlsConfig = testhelpers.TLSClientConfig(certPool, clientCert)
	})

	Context("when request header write times out", func() {
		It("closes the request", func() {
			serverProc = ifrit.Invoke(server)
			addressTable.LookupStub = func(hostname string) []string {
				if hostname == "app-id.internal.local." {
					return []string{"192.168.0.2"}
				}
				return []string{}
			}
			addressTable.IsWarmReturns(true)

			var conn net.Conn
			var err error
			Eventually(func() error {
				conn, err = tls.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port), tlsConfig)
				return err
			}).Should(BeNil())
			defer conn.Close()

			writer := bufio.NewWriter(conn)

			fmt.Fprintf(writer, "GET /v1/registration/app-id.internal.local. HTTP/1.1\r\n")

			// started writing headers
			fmt.Fprintf(writer, "Host: localhost\r\n")
			writer.Flush()

			time.Sleep(300 * time.Millisecond)

			fmt.Fprintf(writer, "User-Agent: CustomClient/1.0\r\n")
			writer.Flush()

			time.Sleep(300 * time.Millisecond)

			// done
			fmt.Fprintf(writer, "\r\n")
			writer.Flush()

			resp := bufio.NewReader(conn)
			_, err = resp.ReadString('\n')
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("EOF"))
		})
	})

	Context("when the lookup succeeds", func() {
		var respBody string

		BeforeEach(func() {
			serverProc = ifrit.Invoke(server)
			addressTable.LookupStub = func(hostname string) []string {
				if hostname == "app-id.internal.local." {
					return []string{"192.168.0.2"}
				}
				return []string{}
			}
			addressTable.IsWarmReturns(true)

			var resp *http.Response
			var err error
			Eventually(func() error {
				resp, err = client.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port))
				return err
			}).Should(BeNil())

			respBodyBytes, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			respBody = string(respBodyBytes)
		})

		AfterEach(func() {
			serverProc.Signal(os.Interrupt)
			Eventually(serverProc.Wait()).Should(Receive())
		})

		It("should return addresses for a give hostname", func() {
			Expect(string(respBody)).To(MatchJSON(`{
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
		})

		It("invokes the dns request recorder", func() {
			Expect(dnsRequestRecorder.RecordRequestCallCount()).To(BeNumerically(">=", 1))
		})

		It("invokes our metrics sender", func() {
			Expect(metricsSender.SendDurationCallCount()).To(BeNumerically(">=", 1))
			name, time := metricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("addressTableLookupTime"))
			Expect(time.String()).ToNot(Equal("0s"))
		})
	})

	Context("when the address table is not warm", func() {
		var (
			resp *http.Response
		)
		BeforeEach(func() {
			serverProc = ifrit.Invoke(server)
			addressTable.IsWarmReturns(false)

			var err error
			Eventually(func() error {
				resp, err = client.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port))
				return err
			}).Should(BeNil())
		})

		It("returns an internal server error", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

			respBodyBytes, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			respBody := string(respBodyBytes)
			Expect(respBody).To(ContainSubstring("address table is not warm"))
		})

		It("logs the error at debug level", func() {
			Expect(testLogger.Logs()).To(HaveLen(2))
			Expect(testLogger.Logs()[1]).To(SatisfyAll(
				LogsWith(lager.DEBUG, "test.failed-request"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("serviceKey", Equal("app-id.internal.local.")),
					HaveKeyWithValue("reason", Equal("address-table-not-warm")),
				)),
			))
		})
	})

	Context("when signaled an interrupt", func() {
		It("shuts down", func() {
			serverProc = ifrit.Invoke(server)

			Eventually(func() error {
				_, err := client.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port))
				return err
			}).Should(BeNil())

			serverProc.Signal(os.Interrupt)
			Eventually(serverProc.Wait()).Should(Receive())
			Eventually(testLogger.LogMessages).Should(ContainElement("test.SDC http server exiting with signal: interrupt"))

			client := testhelpers.NewClient(testhelpers.CertPool(caFile), clientCert)
			_, err := client.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/registration/app-id.internal.local.", port))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection refused"))
		})
	})

	Context("when it is unable to start", func() {
		var conflictingServer *http.Server

		BeforeEach(func() {
			conflictingServer = testhelpers.LaunchConflictingServer(port)
		})

		AfterEach(func() {
			conflictingServer.Close()
			serverProc.Signal(os.Interrupt)
			Eventually(serverProc.Wait()).Should(Receive())
		})

		It("logs and quits", func() {
			serverProc = ifrit.Invoke(server)
			Eventually(serverProc.Wait()).Should(Receive())
			Eventually(testLogger.LogMessages(), 5*time.Second).Should(
				ContainElement(fmt.Sprintf("test.SDC http server exiting with: listen tcp 127.0.0.1:%d: bind: address already in use", port)),
			)
		})
	})
})
