package handlers_test

import (
	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/lager/lagertest"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/store"
	storeFakes "policy-server/store/fakes"
)

var _ = Describe("DestinationDelete", func() {

	var (
		expectedResponseBody []byte
		request              *http.Request
		handler              *handlers.DestinationDelete
		resp                 *httptest.ResponseRecorder
		fakeMetricsSender    *storeFakes.MetricsSender
		fakeStore            *fakes.EgressDestinationStoreDeleter
		fakeMarshaller       *fakes.EgressDestinationMarshaller
		logger               *lagertest.TestLogger
		deletedDestination   store.EgressDestination
	)

	BeforeEach(func() {
		expectedResponseBody = []byte("some-response")

		var err error
		path := "/networking/v1/external/destinations/destguid"
		request, err = http.NewRequest("DELETE", path, nil)
		request.URL.RawQuery = ":id=destguid"
		Expect(err).NotTo(HaveOccurred())

		deletedDestination = store.EgressDestination{
			GUID: "guid-from-store",
		}

		fakeStore = &fakes.EgressDestinationStoreDeleter{}
		fakeStore.DeleteReturns(deletedDestination, nil)

		fakeMarshaller = &fakes.EgressDestinationMarshaller{}
		fakeMarshaller.AsBytesReturns(expectedResponseBody, nil)

		logger = lagertest.NewTestLogger("test")

		fakeMetricsSender = &storeFakes.MetricsSender{}

		errorResponse := &httperror.ErrorResponse{
			MetricsSender: fakeMetricsSender,
		}

		handler = &handlers.DestinationDelete{
			ErrorResponse:           errorResponse,
			EgressDestinationStore:  fakeStore,
			EgressDestinationMapper: fakeMarshaller,
			Logger:                  logger,
		}
		resp = httptest.NewRecorder()
	})

	It("deletes destinations", func() {
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakeStore.DeleteCallCount()).To(Equal(1))
		Expect(fakeStore.DeleteArgsForCall(0)).To(Equal("destguid"))
		Expect(fakeMarshaller.AsBytesCallCount()).To(Equal(1))
		Expect(fakeMarshaller.AsBytesArgsForCall(0)).To(Equal([]store.EgressDestination{deletedDestination}))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	})

	It("returns an error when the store returns an error", func() {
		fakeStore.DeleteReturns(store.EgressDestination{}, errors.New("can't delete"))
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error deleting egress destination"}`))
	})

	It("returns an error when the marshalling deleted destination fails", func() {
		fakeMarshaller.AsBytesReturns(nil, errors.New("can't serialize"))
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error serializing egress destination"}`))
	})
})
