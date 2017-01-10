package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/fakes"
	"policy-server/handlers"
	"policy-server/models"
	"policy-server/uaa_client"

	lfakes "lib/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Policies index handler", func() {
	var (
		allPolicies      []models.Policy
		filteredPolicies []models.Policy
		request          *http.Request
		handler          *handlers.PoliciesIndex
		resp             *httptest.ResponseRecorder
		fakeStore        *fakes.Store
		fakePolicyFilter *fakes.PolicyFilter
		logger           *lagertest.TestLogger
		marshaler        *lfakes.Marshaler
		token            uaa_client.CheckTokenResponse
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
		}, {
			Source: models.Source{ID: "yet-another-app-guid"},
			Destination: models.Destination{
				ID:       "yet-another-app-guid",
				Protocol: "udp",
				Port:     5555,
			},
		}}

		filteredPolicies = []models.Policy{{
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
		fakePolicyFilter = &fakes.PolicyFilter{}
		fakePolicyFilter.FilterPoliciesStub = func(policies []models.Policy, userToken uaa_client.CheckTokenResponse) ([]models.Policy, error) {
			return filteredPolicies, nil
		}
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.PoliciesIndex{
			Logger:       logger,
			Store:        fakeStore,
			Marshaler:    marshaler,
			PolicyFilter: fakePolicyFilter,
		}

		token = uaa_client.CheckTokenResponse{
			Scope:    []string{"some-scope", "some-other-scope"},
			UserID:   "some-user-id",
			UserName: "some-user",
		}
		resp = httptest.NewRecorder()
	})

	It("returns all the policies, but does not include the tags", func() {
		expectedResponseJSON := `{
			"total_policies": 2,
			"policies": [
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
		handler.ServeHTTP(resp, request, token)

		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
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

		It("filters only those policies which contain that id", func() {
			var filteredByID []models.Policy
			filteredByID = append(filteredByID, allPolicies[0], allPolicies[2])

			handler.ServeHTTP(resp, request, token)

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
			policies, userToken := fakePolicyFilter.FilterPoliciesArgsForCall(0)
			Expect(policies).To(Equal(filteredByID))
			Expect(userToken).To(Equal(token))
			Expect(resp.Code).To(Equal(http.StatusOK))
		})

		It("calls the policy filter with an empty list when the id list is empty", func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/external/policies?id=", nil)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(resp, request, token)
			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))

			policies, userToken := fakePolicyFilter.FilterPoliciesArgsForCall(0)
			Expect(policies).To(Equal([]models.Policy{}))
			Expect(userToken).To(Equal(token))

			Expect(resp.Code).To(Equal(http.StatusOK))
		})
	})

	Context("when the store throws an error", func() {
		BeforeEach(func() {
			fakeStore.AllReturns(nil, errors.New("banana"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, token)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database read failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, token)
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
			handler.ServeHTTP(resp, request, token)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database marshaling failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, token)
			Expect(logger).To(gbytes.Say("marshal-failed.*grapes"))
		})
	})

	Context("when the policy guard throws an error", func() {
		BeforeEach(func() {
			fakePolicyFilter.FilterPoliciesReturns(nil, errors.New("banana"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, token)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "filter policies failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, token)
			Expect(logger).To(gbytes.Say("filter-policies-failed.*banana"))
		})
	})
})
