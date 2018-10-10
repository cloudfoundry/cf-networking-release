package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/uaa_client"

	"policy-server/store"
	storeFakes "policy-server/store/fakes"

	"bytes"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Destinations update handler", func() {
	var (
		expectedResponseBody  []byte
		request               *http.Request
		handler               *handlers.DestinationsUpdate
		resp                  *httptest.ResponseRecorder
		fakeMetricsSender     *storeFakes.MetricsSender
		fakeStore             *fakes.EgressDestinationStoreUpdater
		fakeMarshaller        *fakes.EgressDestinationMarshaller
		logger                *lagertest.TestLogger
		updatedDestinations   []store.EgressDestination
		requestedDestinations []store.EgressDestination
		token                 uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		expectedResponseBody = []byte("some-response")

		requestBody := `{
			"destinations": [
				{
					"id": "some-guid-id-1
					"name": "my service",
				  "description": "my service is a great service",	
				  "ips": [{"start": "21.30.35.9", "end": "72.30.35.9"}],
				  "ports": [{"start": 8080, "end": 8080}],
				  "protocol":"tcp"
				},
				{
					"id": "some-guid-id-2
					"name": "my service 2",
				  "description": "my service is not so great service",	
				  "ips": [{"start": "21.30.35.9", "end": "72.30.35.9"}],
				  "ports": [{"start": 8080, "end": 8080}],
				  "protocol":"tcp"
				}
			]
		}`

		var err error
		request, err = http.NewRequest("PUT", "/networking/v1/external/destinations", bytes.NewBuffer([]byte(requestBody)))
		Expect(err).NotTo(HaveOccurred())

		updatedDestination1 := store.EgressDestination{GUID: "created-one"}
		updatedDestination2 := store.EgressDestination{GUID: "created-two"}
		updatedDestinations = []store.EgressDestination{updatedDestination1, updatedDestination2}

		fakeStore = &fakes.EgressDestinationStoreUpdater{}
		fakeStore.UpdateReturns(updatedDestinations, nil)

		fakeMarshaller = &fakes.EgressDestinationMarshaller{}

		requestedDestinations = []store.EgressDestination{
			{GUID: "req-one"},
			{GUID: "req-two"},
		}
		fakeMarshaller.AsEgressDestinationsReturns(requestedDestinations, nil)

		fakeMarshaller.AsBytesReturns(expectedResponseBody, nil)

		logger = lagertest.NewTestLogger("test")

		fakeMetricsSender = &storeFakes.MetricsSender{}

		errorResponse := &httperror.ErrorResponse{
			MetricsSender: fakeMetricsSender,
		}

		handler = &handlers.DestinationsUpdate{
			ErrorResponse:           errorResponse,
			EgressDestinationStore:  fakeStore,
			EgressDestinationMapper: fakeMarshaller,
			Logger:                  logger,
		}
		resp = httptest.NewRecorder()

		token = uaa_client.CheckTokenResponse{
			Scope:    []string{"some-scope", "network.admin"},
			UserID:   "some-user-id",
			UserName: "some-user",
		}
	})

	It("updates destinations", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(fakeStore.UpdateCallCount()).To(Equal(1))
		Expect(fakeStore.UpdateArgsForCall(0)).To(Equal(requestedDestinations))
		Expect(fakeMarshaller.AsBytesCallCount()).To(Equal(1))
		Expect(fakeMarshaller.AsBytesArgsForCall(0)).To(Equal(updatedDestinations))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	})

	It("returns an error when the request body can't be read", func() {
		request.Body = &failingReader{}
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error reading request"}`))
	})

	It("returns an error when the mapper returns an error", func() {
		fakeMarshaller.AsEgressDestinationsReturns([]store.EgressDestination{}, errors.New("whoa"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusBadRequest))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error parsing egress destination: whoa"}`))
	})

	It("returns an error when the store returns an error", func() {
		fakeStore.UpdateReturns([]store.EgressDestination{}, errors.New("oh noes"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error updating egress destination"}`))
	})

	It("returns an duplicate name error when the store returns duplicate entry error", func() {
		fakeStore.UpdateReturns([]store.EgressDestination{}, errors.New("blah blah: duplicate name error: blah"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusBadRequest))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error updating egress destination: blah blah: duplicate name error: blah"}`))
	})

	It("returns an error when marshalling the updated destination fails", func() {
		fakeMarshaller.AsBytesReturns(nil, errors.New("can't serialize"))
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error serializing egress destinations"}`))
	})

	It("returns an error when requested update destination is missing an ID", func() {
		requestedDestinationsWithMissingGUID := []store.EgressDestination{
			{GUID: "req-one"},
			{Name: "a name is not a GUID"},
		}
		fakeMarshaller.AsEgressDestinationsReturns(requestedDestinationsWithMissingGUID, nil)

		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusBadRequest))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "destination id not found on request"}`))
	})

	It("returns an error when requested update destination is not in the database", func() {
		fakeStore.UpdateReturns([]store.EgressDestination{}, errors.New("blah blah: destination GUID not found: blah blah"))

		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
		Expect(resp.Code).To(Equal(http.StatusNotFound))
		Expect(resp.Body.Bytes()).To(MatchJSON(`{"error": "error updating egress destination: blah blah: destination GUID not found: blah blah"}`))
	})
})
