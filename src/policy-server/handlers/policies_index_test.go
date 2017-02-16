package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/models"
	"policy-server/uaa_client"

	lfakes "lib/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Policies index handler", func() {
	var (
		allPolicies       []models.Policy
		byGuidsPolicies   []models.Policy
		filteredPolicies  []models.Policy
		request           *http.Request
		handler           *handlers.PoliciesIndex
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.Store
		fakePolicyFilter  *fakes.PolicyFilter
		fakeErrorResponse *fakes.ErrorResponse
		logger            *lagertest.TestLogger
		marshaler         *lfakes.Marshaler
		token             uaa_client.CheckTokenResponse
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

		byGuidsPolicies = []models.Policy{{
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

		filteredPolicies = []models.Policy{{
			Source: models.Source{ID: "some-app-guid", Tag: "some-tag"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Tag:      "some-other-tag",
				Protocol: "tcp",
				Port:     8080,
			},
		}}

		var err error
		request, err = http.NewRequest("GET", "/networking/v0/external/policies", nil)
		Expect(err).NotTo(HaveOccurred())

		marshaler = &lfakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		fakeStore = &fakes.Store{}
		fakeStore.AllReturns(allPolicies, nil)
		fakeStore.ByGuidsReturns(byGuidsPolicies, nil)
		fakeErrorResponse = &fakes.ErrorResponse{}
		fakePolicyFilter = &fakes.PolicyFilter{}
		fakePolicyFilter.FilterPoliciesStub = func(policies []models.Policy, userToken uaa_client.CheckTokenResponse) ([]models.Policy, error) {
			return filteredPolicies, nil
		}
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.PoliciesIndex{
			Logger:        logger,
			Store:         fakeStore,
			Marshaler:     marshaler,
			PolicyFilter:  fakePolicyFilter,
			ErrorResponse: fakeErrorResponse,
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
			"total_policies": 1,
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
			var err error
			request, err = http.NewRequest("GET", "/networking/v0/external/policies?id=some-app-guid,yet-another-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("filters on only those policies returned by ByGuids", func() {
			handler.ServeHTTP(resp, request, token)

			Expect(fakeStore.ByGuidsCallCount()).To(Equal(1))
			srcGuids, dstGuids := fakeStore.ByGuidsArgsForCall(0)
			Expect(srcGuids).To(ConsistOf([]string{"some-app-guid", "yet-another-app-guid"}))
			Expect(dstGuids).To(ConsistOf([]string{"some-app-guid", "yet-another-app-guid"}))
			Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
			policies, userToken := fakePolicyFilter.FilterPoliciesArgsForCall(0)
			Expect(policies).To(Equal(byGuidsPolicies))
			Expect(userToken).To(Equal(token))
			Expect(resp.Code).To(Equal(http.StatusOK))
		})

		Context("when the id list is empty", func() {
			It("filters on only those policies returned by ByGuids", func() {
				var err error
				request, err = http.NewRequest("GET", "/networking/v0/external/policies?id=", nil)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(resp, request, token)
				Expect(fakeStore.ByGuidsCallCount()).To(Equal(1))
				srcGuids, destGuids := fakeStore.ByGuidsArgsForCall(0)
				Expect(srcGuids).To(Equal([]string{""}))
				Expect(destGuids).To(Equal([]string{""}))
				Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
				policies, userToken := fakePolicyFilter.FilterPoliciesArgsForCall(0)
				Expect(policies).To(Equal(byGuidsPolicies))
				Expect(userToken).To(Equal(token))

				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Context("when the store throws an error", func() {
		BeforeEach(func() {
			fakeStore.AllReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-index"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	Context("when the policy cannot be marshaled", func() {
		BeforeEach(func() {
			marshaler.MarshalStub = func(interface{}) ([]byte, error) {
				return nil, errors.New("grapes")
			}
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("grapes"))
			Expect(message).To(Equal("policies-index"))
			Expect(description).To(Equal("database marshaling failed"))
		})
	})

	Context("when the policy filter throws an error", func() {
		BeforeEach(func() {
			fakePolicyFilter.FilterPoliciesReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-index"))
			Expect(description).To(Equal("filter policies failed"))
		})
	})
})
