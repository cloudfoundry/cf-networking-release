package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/fakes"
	"policy-server/handlers"
	"policy-server/models"

	lfakes "lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Policies index handler", func() {
	var (
		allPolicies []models.Policy
		request     *http.Request
		handler     *handlers.PoliciesIndex
		resp        *httptest.ResponseRecorder
		fakeStore   *fakes.Store
		logger      *lagertest.TestLogger
		marshaler   *lfakes.Marshaler
	)

	BeforeEach(func() {
		allPolicies = []models.Policy{{
			Source: models.Source{ID: "some-app-guid", Tag: "some-tag"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Tag:      "some-other-tag",
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

		var err error
		request, err = http.NewRequest("GET", "/networking/v0/external/policies", nil)
		Expect(err).NotTo(HaveOccurred())

		marshaler = &lfakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		fakeStore = &fakes.Store{}
		fakeStore.AllReturns(allPolicies, nil)
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.PoliciesIndex{
			Logger:    logger,
			Store:     fakeStore,
			Marshaler: marshaler,
		}
		resp = httptest.NewRecorder()
	})

	It("returns all the policies, but does not include the tags", func() {
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
		handler.ServeHTTP(resp, request, "")

		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body).To(MatchJSON(expectedResponseJSON))
	})

	Context("when a list of ids is provided as a query parameter", func() {
		BeforeEach(func() {
			allPolicies = []models.Policy{{
				Source: models.Source{ID: "some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			},
				{
					Source: models.Source{ID: "another-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "udp",
						Port:     1234,
					},
				},
				{
					Source: models.Source{ID: "another-app-guid"},
					Destination: models.Destination{
						ID:       "yet-another-app-guid",
						Protocol: "udp",
						Port:     5678,
					},
				},
			}
			fakeStore.AllReturns(allPolicies, nil)

			var err error
			request, err = http.NewRequest("GET", "/networking/v0/external/policies?id=some-app-guid,yet-another-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns only those policies which contain that id", func() {
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
					"id": "yet-another-app-guid",
					"protocol": "udp",
					"port": 5678
				}
			}
        ]}`
			handler.ServeHTTP(resp, request, "")

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body).To(MatchJSON(expectedResponseJSON))
		})

		It("returns an empty list when the id list is empty", func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/external/policies?id=", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(resp, request, "")
			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body).To(MatchJSON(`{"policies": []}`))
		})
	})

	Context("when the store throws an error", func() {
		BeforeEach(func() {
			fakeStore.AllReturns(nil, errors.New("banana"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, "")

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database read failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(logger).To(gbytes.Say("store-list-policies-failed.*banana"))
		})
	})

	Context("when the policy cannot be marshaled", func() {
		BeforeEach(func() {
			marshaler.MarshalStub = func(interface{}) ([]byte, error) {
				return nil, errors.New("grapes")
			}
		})

		It("responds with 500 and returns a descriptive error", func() {
			handler.ServeHTTP(resp, request, "")

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database marshaling failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(logger).To(gbytes.Say("marshal-failed.*grapes"))
		})
	})
})
