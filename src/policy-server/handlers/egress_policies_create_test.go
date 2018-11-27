package handlers_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	storeFakes "policy-server/store/fakes"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"

	"policy-server/uaa_client"

	"policy-server/store"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPoliciesCreate", func() {
	var (
		expectedStoreEgressPolicies []store.EgressPolicy
		fakeMapper                  *fakes.EgressPolicyMapper
		fakeStore                   *fakes.EgressPolicyStore
		logger                      *lagertest.TestLogger
		fakeMetricsSender           *storeFakes.MetricsSender
		handler                     *handlers.EgressPolicyCreate
		resp                        *httptest.ResponseRecorder
		request                     *http.Request
		requestBody                 string
		responseBody                string
		token                       uaa_client.CheckTokenResponse
		createdPolicies             []store.EgressPolicy
	)

	BeforeEach(func() {
		fakeStore = &fakes.EgressPolicyStore{}
		fakeMapper = &fakes.EgressPolicyMapper{}

		fakeMetricsSender = &storeFakes.MetricsSender{}
		errorResponse := &httperror.ErrorResponse{
			MetricsSender: fakeMetricsSender,
		}

		logger = lagertest.NewTestLogger("test")

		handler = &handlers.EgressPolicyCreate{
			Store:         fakeStore,
			Mapper:        fakeMapper,
			ErrorResponse: errorResponse,
			Logger:        logger,
		}

		var err error
		requestBody = `{
			"egress_policies": [
				{
					"source": { "id": "AN-APP-GUID", "type": "app" },
					"destination": {"id": "A-DEST-GUID" },
					"app_lifecycle": "staging"
				}
			]
		}`

		expectedStoreEgressPolicies = []store.EgressPolicy{
			{
				Source:      store.EgressSource{ID: "AN-APP-GUID"},
				Destination: store.EgressDestination{GUID: "A-DEST-GUID"},
				AppLifecycle: "staging",
			},
		}

		createdPolicies = []store.EgressPolicy{
			{
				ID: "abc-123",
			},
		}
		fakeStore.CreateReturns(createdPolicies, nil)

		fakeMapper.AsStoreEgressPolicyReturns(expectedStoreEgressPolicies, nil)
		responseBody = `{
			"egress_policies": [
				{
					"id": "policy-guid",
					"source": { "id": "AN-APP-GUID", "type": "app" },
					"destination": {"id": "A-DEST-GUID" },
					"app_lifecycle": "staging"
				}
			]
		}`

		fakeMapper.AsBytesReturns([]byte(responseBody), nil)

		request, err = http.NewRequest("POST", "/networking/v1/external/egress_policies", bytes.NewBuffer([]byte(requestBody)))
		Expect(err).NotTo(HaveOccurred())

		resp = httptest.NewRecorder()

		token = uaa_client.CheckTokenResponse{Scope: []string{"some-scope"}}
	})

	It("creates an egress policy", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(fakeMapper.AsStoreEgressPolicyCallCount()).To(Equal(1))
		policies := fakeMapper.AsStoreEgressPolicyArgsForCall(0)
		Expect(string(policies)).To(Equal(requestBody))

		Expect(fakeStore.CreateCallCount()).To(Equal(1))
		storePolicies := fakeStore.CreateArgsForCall(0)
		Expect(storePolicies).To(Equal(expectedStoreEgressPolicies))

		Expect(fakeMapper.AsBytesCallCount()).To(Equal(1))
		Expect(fakeMapper.AsBytesArgsForCall(0)).To(Equal(createdPolicies))
	})

	It("returns a response that includes the guid for the created policy", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(body)).To(Equal(responseBody))
	})

	It("returns a 400 when the request body can not be read", func() {
		request.Body = &failingReader{}
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(resp.Code).To(Equal(http.StatusBadRequest))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error reading request"}`))
	})

	It("returns an error when the store returns an error", func() {
		fakeStore.CreateReturns(nil, errors.New("can't create"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error creating egress policy"}`))
	})

	It("returns an error when parsing the request returns an error", func() {
		fakeMapper.AsStoreEgressPolicyReturns(nil, errors.New("didn't go well"))

		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusBadRequest))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error parsing egress policies: didn't go well"}`))
	})

	It("returns an error response when marshalling the response returns an error", func() {
		fakeMapper.AsBytesReturns(nil, errors.New("didn't go well"))

		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error serializing response"}`))
	})
})
