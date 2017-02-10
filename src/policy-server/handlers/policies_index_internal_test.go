package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/models"

	lfakes "lib/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PoliciesIndexInternal", func() {
	var (
		handler           *handlers.PoliciesIndexInternal
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.Store
		fakeErrorResponse *fakes.ErrorResponse
		logger            *lagertest.TestLogger
		marshaler         *lfakes.Marshaler
	)

	BeforeEach(func() {
		allPolicies := []models.Policy{{
			Source: models.Source{ID: "some-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Port:     8080,
			},
		}, {
			Source: models.Source{ID: "another-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "udp",
				Port:     1234,
			},
		}}

		marshaler = &lfakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		fakeStore = &fakes.Store{}
		fakeStore.AllReturns(allPolicies, nil)
		logger = lagertest.NewTestLogger("test")
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.PoliciesIndexInternal{
			Logger:        logger,
			Store:         fakeStore,
			Marshaler:     marshaler,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()
	})

	It("it returns only policies that match the filter", func() {
		expectedResponseJSON := `{"policies": [
				{
					"source": {
						"id": "some-app-guid"
					},
					"destination": {
						"id": "some-other-app-guid",
						"protocol": "tcp",
						"port": 8080
					}
				}
			]}`
		request, err := http.NewRequest("GET", "/networking/v0/internal/policies?id=some-app-guid", nil)
		Expect(err).NotTo(HaveOccurred())
		handler.ServeHTTP(resp, request)

		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body).To(MatchJSON(expectedResponseJSON))
	})

	Context("when there are no policies", func() {
		It("returns an empty set", func() {
			fakeStore.AllReturns([]models.Policy{}, nil)
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			request.RemoteAddr = "some-host:some-port"

			handler.ServeHTTP(resp, request)
			Expect(logger).To(gbytes.Say("internal request made to list policies.*RemoteAddr.*some-host:some-port.*URL.*/networking/v0/internal/policies"))

			Expect(resp.Body).To(MatchJSON(`{ "policies": [] }`))
		})
	})

	Context("when there are policies and no filter is passed", func() {
		It("it returns all of them", func() {
			expectedResponseJSON := `{"policies": [
				{
					"source": {
						"id": "some-app-guid"
					},
					"destination": {
						"id": "some-other-app-guid",
						"protocol": "tcp",
						"port": 8080
					}
				},
				{
					"source": {
						"id": "another-app-guid"
					},
					"destination": {
						"id": "some-other-app-guid",
						"protocol": "udp",
						"port": 1234
					}
				}
			]}`
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(resp, request)

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body).To(MatchJSON(expectedResponseJSON))

		})
	})

	Context("when the store throws an error", func() {
		var request *http.Request

		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			fakeStore.AllReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(resp, request)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-index-internal"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	Context("when the policy cannot be marshaled", func() {
		var request *http.Request

		BeforeEach(func() {
			marshaler.MarshalStub = func(interface{}) ([]byte, error) {
				return nil, errors.New("grapes")
			}

			var err error
			request, err = http.NewRequest("get", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("grapes"))
			Expect(message).To(Equal("policies-index-internal"))
			Expect(description).To(Equal("database marshaling failed"))
		})
	})
})
