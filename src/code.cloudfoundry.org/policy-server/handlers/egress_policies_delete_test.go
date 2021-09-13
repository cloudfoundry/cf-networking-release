package handlers_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/handlers/fakes"
	"code.cloudfoundry.org/policy-server/store"
	storeFakes "code.cloudfoundry.org/policy-server/store/fakes"
	"code.cloudfoundry.org/policy-server/uaa_client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPoliciesDelete", func() {
	var (
		fakeMapper        *fakes.EgressPolicyMapper
		fakeStore         *fakes.EgressPolicyStore
		logger            *lagertest.TestLogger
		fakeMetricsSender *storeFakes.MetricsSender
		handler           *handlers.EgressPolicyDelete
		resp              *httptest.ResponseRecorder
		request           *http.Request
		responseBody      string
		token             uaa_client.CheckTokenResponse
		deletedPolicies   []store.EgressPolicy
	)

	BeforeEach(func() {
		fakeStore = &fakes.EgressPolicyStore{}
		fakeMapper = &fakes.EgressPolicyMapper{}

		fakeMetricsSender = &storeFakes.MetricsSender{}
		errorResponse := &httperror.ErrorResponse{
			MetricsSender: fakeMetricsSender,
		}

		logger = lagertest.NewTestLogger("test")

		handler = &handlers.EgressPolicyDelete{
			Store:         fakeStore,
			Mapper:        fakeMapper,
			ErrorResponse: errorResponse,
			Logger:        logger,
		}

		deletedPolicies = []store.EgressPolicy{
			{
				ID: "abc-123",
			},
		}
		fakeStore.DeleteReturns(deletedPolicies, nil)

		responseBody = `{
			"egress_policies": [
				{
					"id": "abc-123",
					"source": { "id": "AN-APP-GUID", "type": "app" },
					"destination": {"id": "A-DEST-GUID" }
				}
			]
		}`

		fakeMapper.AsBytesReturns([]byte(responseBody), nil)

		var err error
		request, err = http.NewRequest("DELETE", "/networking/v1/external/egress_policies/abc-123", nil)
		request.URL.RawQuery = ":id=abc-123"
		Expect(err).NotTo(HaveOccurred())

		resp = httptest.NewRecorder()

		token = uaa_client.CheckTokenResponse{Scope: []string{"some-scope"}}
	})

	It("deletes an egress policy", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(fakeStore.DeleteCallCount()).To(Equal(1))
		guidToBeDeleted := fakeStore.DeleteArgsForCall(0)
		Expect(guidToBeDeleted).To(ConsistOf("abc-123"))

		Expect(fakeMapper.AsBytesCallCount()).To(Equal(1))
		Expect(fakeMapper.AsBytesArgsForCall(0)).To(Equal(deletedPolicies))

		Expect(resp.Code).To(Equal(http.StatusOK))
	})

	It("returns a response that includes the deleted policy", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(body)).To(Equal(responseBody))
	})

	It("returns an error when the store returns an error", func() {
		fakeStore.DeleteReturns([]store.EgressPolicy{}, errors.New("can't create"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error deleting egress policy"}`))
	})

	It("returns an error the mapper cannot serialize the output", func() {
		fakeMapper.AsBytesReturns(nil, errors.New("didn't go well"))

		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error serializing response"}`))
	})
})
