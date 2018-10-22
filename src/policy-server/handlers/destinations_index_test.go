package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	"policy-server/uaa_client"

	"policy-server/store"
	storeFakes "policy-server/store/fakes"

	"errors"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Destinations index handler", func() {
	var (
		allDestinations      []store.EgressDestination
		expectedResponseBody []byte
		request              *http.Request
		handler              *handlers.DestinationsIndex
		resp                 *httptest.ResponseRecorder
		fakeMetricsSender    *storeFakes.MetricsSender
		fakeStore            *fakes.EgressDestinationStoreLister
		fakeMapper           *fakes.EgressDestinationMarshaller
		logger               *lagertest.TestLogger
		token                uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		expectedResponseBody = []byte("some-errorResponse")

		var err error
		request, err = http.NewRequest("GET", "/networking/v1/external/destinations", nil)
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.EgressDestinationStoreLister{}
		fakeStore.AllReturns(allDestinations, nil)

		fakeMapper = &fakes.EgressDestinationMarshaller{}
		fakeMapper.AsBytesReturns(expectedResponseBody, nil)

		logger = lagertest.NewTestLogger("test")

		fakeMetricsSender = &storeFakes.MetricsSender{}

		errorResponse := &httperror.ErrorResponse{
			MetricsSender: fakeMetricsSender,
		}

		handler = &handlers.DestinationsIndex{
			ErrorResponse:           errorResponse,
			EgressDestinationMapper: fakeMapper,
			EgressDestinationStore:  fakeStore,
			Logger:                  logger,
		}

		token = uaa_client.CheckTokenResponse{
			Scope:    []string{"some-scope", "some-other-scope"},
			UserID:   "some-user-id",
			UserName: "some-user",
		}
		resp = httptest.NewRecorder()
	})

	It("returns all the destinations", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(fakeMapper.AsBytesCallCount()).To(Equal(1))
		Expect(fakeMapper.AsBytesArgsForCall(0)).To(Equal(allDestinations))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	})

	It("returns an error when the store returns an error", func() {
		fakeStore.AllReturns(nil, errors.New("things went askew"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error getting egress destinations"}`))
	})

	It("returns an error when the mapper returns an error", func() {
		fakeMapper.AsBytesReturns(nil, errors.New("things went askew"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error mapping egress destinations"}`))
	})

	Context("when the logger isn't on the request context", func() {
		It("still works", func() {
			MakeRequestWithAuth(handler.ServeHTTP, resp, request, token)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
		})
	})

	Context("when the id query parameter is passed", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v1/external/destinations?id=some-guid,some-guid-2", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("return only the destination with that guid", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
			Expect(fakeStore.AllCallCount()).To(Equal(0))
			Expect(fakeStore.GetByGUIDCallCount()).To(Equal(1))
			Expect(fakeStore.GetByGUIDArgsForCall(0)).To(Equal([]string{"some-guid", "some-guid-2"}))
		})
	})

	Context("when the name query parameter is passed", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v1/external/destinations?name=some-name,some-name-2", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("return only the destination with that name", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
			Expect(fakeStore.AllCallCount()).To(Equal(0))
			Expect(fakeStore.GetByGUIDCallCount()).To(Equal(0))
			Expect(fakeStore.GetByNameArgsForCall(0)).To(Equal([]string{"some-name", "some-name-2"}))
		})
	})
})
