package handlers_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/store"
	storeFakes "policy-server/store/fakes"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPoliciesIndex", func() {
	var (
		fakeMapper        *fakes.EgressPolicyMapper
		fakeStore         *fakes.EgressPolicyStore
		logger            *lagertest.TestLogger
		fakeMetricsSender *storeFakes.MetricsSender
		handler           *handlers.EgressPolicyIndex
		resp              *httptest.ResponseRecorder
		request           *http.Request
		responseBody      string
		token             uaa_client.CheckTokenResponse
		policies          []store.EgressPolicy
	)

	BeforeEach(func() {
		fakeStore = &fakes.EgressPolicyStore{}
		fakeMapper = &fakes.EgressPolicyMapper{}

		fakeMetricsSender = &storeFakes.MetricsSender{}
		errorResponse := &httperror.ErrorResponse{
			MetricsSender: fakeMetricsSender,
		}

		logger = lagertest.NewTestLogger("test")

		handler = &handlers.EgressPolicyIndex{
			Store:         fakeStore,
			Mapper:        fakeMapper,
			ErrorResponse: errorResponse,
			Logger:        logger,
		}

		policies = []store.EgressPolicy{
			{
				ID: "abc-123",
			},
		}
		responseBody = `{
			"egress_policies": [
				{
					"id": "abc-123",
					"source": { "id": "AN-APP-GUID", "type": "app" },
					"destination": {"id": "A-DEST-GUID" }
				}
			]
		}`

		fakeMapper.AsBytesWithPopulatedDestinationsReturns([]byte(responseBody), nil)

		var err error
		request, err = http.NewRequest("GET", "/networking/v1/external/egress_policies", nil)
		Expect(err).NotTo(HaveOccurred())

		resp = httptest.NewRecorder()

		token = uaa_client.CheckTokenResponse{Scope: []string{"some-scope"}}
	})

	It("lists egress policies", func() {
		fakeStore.GetByFilterReturns(policies, nil)
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(fakeStore.GetByFilterCallCount()).To(Equal(1))

		Expect(fakeMapper.AsBytesWithPopulatedDestinationsCallCount()).To(Equal(1))
		Expect(fakeMapper.AsBytesWithPopulatedDestinationsArgsForCall(0)).To(Equal(policies))

		Expect(resp.Code).To(Equal(http.StatusOK))
	})

	It("returns a response that includes the deleted policy", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(body)).To(Equal(responseBody))
	})

	It("returns an error when the store returns an error", func() {
		fakeStore.GetByFilterReturns([]store.EgressPolicy{}, errors.New("can't create"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error listing egress policies"}`))
	})

	It("returns an error the mapper cannot serialize the output", func() {
		fakeMapper.AsBytesWithPopulatedDestinationsReturns(nil, errors.New("didn't go well"))

		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error serializing response"}`))
	})

	Context("when the query parameters are passed", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", `/networking/v1/external/egress_policies?SourceIDs=abc-123&SourceTypes=outerSpace&DestinationIDs=xyz789,helloguid&DestinationNames=moon%20walk`, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("return only the destination with that guid", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
			Expect(fakeStore.GetByFilterCallCount()).To(Equal(1))
			sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles := fakeStore.GetByFilterArgsForCall(0)
			Expect(sourceIds).To(Equal([]string{"abc-123"}))
			Expect(sourceTypes).To(Equal([]string{"outerSpace"}))
			Expect(destinationIds).To(Equal([]string{"xyz789", "helloguid"}))
			Expect(destinationNames).To(Equal([]string{"moon walk"}))
			Expect(appLifecycles).To(Equal([]string{}))
		})
	})
})
