package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/bosh-dns-adapter/handlers"
	"code.cloudfoundry.org/bosh-dns-adapter/handlers/fakes"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetIP", func() {
	var (
		getIP handlers.GetIP

		fakeSDCClient     *fakes.SDCClient
		fakeCopilotClient *fakes.CopilotClient
		fakeMetricsSender *fakes.MetricsSender

		resp    *httptest.ResponseRecorder
		request *http.Request
		logger  *lagertest.TestLogger
	)

	BeforeEach(func() {
		fakeSDCClient = &fakes.SDCClient{}
		fakeSDCClient.IPsReturns([]string{"192.168.0.1"}, nil)

		fakeCopilotClient = &fakes.CopilotClient{}
		fakeCopilotClient.IPReturns("", errors.New("fake not initialized"))

		fakeMetricsSender = &fakes.MetricsSender{}

		logger = lagertest.NewTestLogger("get ip handler test logger")

		getIP = handlers.GetIP{
			SDCClient:                  fakeSDCClient,
			CopilotClient:              fakeCopilotClient,
			MetricsSender:              fakeMetricsSender,
			InternalServiceMeshDomains: []string{"app-id.istio.internal.local."},
			Logger:                     logger,
		}

		resp = httptest.NewRecorder()
	})

	Context("when the user requests an A record for a given hostname", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "?type=1&name=app.example.com.", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("return an A record response with a list of ips", func() {
			getIP.ServeHTTP(resp, request)

			Expect(fakeSDCClient.IPsCallCount()).To(Equal(1))
			Expect(fakeSDCClient.IPsArgsForCall(0)).To(Equal("app.example.com."))

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON(`{
					"Status": 0,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app.example.com.",
							"type": 1
						}
					],
					"Answer":
					[
						{
							"name": "app.example.com.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when the user provides only a hostname", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "?name=app.example.com.", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an A record with a list of ips", func() {
			getIP.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON(`{
					"Status": 0,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app.example.com.",
							"type": 1
						}
					],
					"Answer":
					[
						{
							"name": "app.example.com.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when the user makes a requests without providing the hostname", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "?type=1", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a http 400 status", func() {
			getIP.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{
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

	Context("when requesting anything but an A record", func() {
		It("should return a successful response with no answers", func() {
			request, err := http.NewRequest("GET", "?type=16&name=app-id.internal.local.", nil)
			Expect(err).ToNot(HaveOccurred())

			getIP.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON(`{
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

	Context("when the sdc client returns an error", func() {
		var request *http.Request
		BeforeEach(func() {
			fakeSDCClient.IPsReturns(nil, errors.New("failed to get ips"))
			var err error
			request, err = http.NewRequest("GET", "?type=1&name=app-id.internal.local.", nil)
			Expect(err).To(Succeed())
		})

		It("returns a http 500 status", func() {
			getIP.ServeHTTP(resp, request)
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})

		It("logs an error", func() {
			getIP.ServeHTTP(resp, request)
			Expect(logger.LogMessages()).To(ContainElement(ContainSubstring("could not connect to service discovery controller")))
			Expect(logger.Logs()[0].Data["error"]).To(Equal("failed to get ips"))
		})

		It("increments the DNSRequestFailures metric counter", func() {
			getIP.ServeHTTP(resp, request)
			Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("DNSRequestFailures"))
		})
	})

	Context("when internal service mesh domain", func() {
		It("should return a http 200 status", func() {
			fakeCopilotClient.IPReturns("127.1.2.3", nil)
			request, err := http.NewRequest("GET", "?type=1&name=app-id.istio.internal.local.", nil)
			Expect(err).ToNot(HaveOccurred())

			getIP.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON(`{
					"Status": 0,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app-id.istio.internal.local.",
							"type": 1
						}
					],
					"Answer":
					[
						{
							"name": "app-id.istio.internal.local.",
							"type": 1,
							"TTL":  0,
							"data": "127.1.2.3"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))

			Expect(fakeCopilotClient.IPCallCount()).To(Equal(1))
			Expect(fakeCopilotClient.IPArgsForCall(0)).To(Equal("app-id.istio.internal.local."))
		})

		Context("when the copilot client returns an error", func() {
			var request *http.Request
			BeforeEach(func() {
				fakeCopilotClient.IPReturns("", errors.New("copilot issues"))

				var err error
				request, err = http.NewRequest("GET", "?type=1&name=app-id.istio.internal.local.", nil)
				Expect(err).To(Succeed())
			})

			It("returns an error response", func() {
				getIP.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})

			It("returns a http 500 status", func() {
				getIP.ServeHTTP(resp, request)
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})

			It("logs an error", func() {
				getIP.ServeHTTP(resp, request)
				Expect(logger.LogMessages()).To(ContainElement(ContainSubstring("could not connect to copilot")))
				Expect(logger.Logs()[0].Data["error"]).To(Equal("copilot issues"))
			})

			It("increments the DNSRequestFailures metric counter", func() {
				getIP.ServeHTTP(resp, request)
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("DNSRequestFailures"))
			})
		})
	})

	It("logs on success", func() {
		fakeCopilotClient.IPReturns("127.1.2.3", nil)

		request, err := http.NewRequest("GET", "?type=1&name=app-id.istio.internal.local.", nil)
		Expect(err).ToNot(HaveOccurred())

		getIP.ServeHTTP(resp, request)
		Expect(resp.Code).To(Equal(http.StatusOK))

		Expect(logger.LogMessages()).To(ContainElement(ContainSubstring("success")))
		Expect(logger.Logs()[1].Data["ips"]).To(Equal("127.1.2.3"))
		Expect(logger.Logs()[1].Data["service-name"]).To(Equal("app-id.istio.internal.local."))
		Expect(logger.Logs()[1].Data["duration-ns"]).To(BeNumerically(">", 0))
	})
})
