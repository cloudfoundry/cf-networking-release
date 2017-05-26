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

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesCleanup", func() {
	var (
		request           *http.Request
		handler           *handlers.PoliciesCleanup
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
		fakePolicyCleaner *fakes.PolicyCleaner
		fakeMarshaler     *hfakes.Marshaler
		fakeErrorResponse *fakes.ErrorResponse
		policies          []models.Policy
		tokenData         uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		policies = []models.Policy{{
			Source: models.Source{ID: "live-guid", Tag: "tag"},
			Destination: models.Destination{
				ID:       "dead-guid",
				Tag:      "tag",
				Protocol: "tcp",
				Port:     8080,
			},
		}}

		logger = lagertest.NewTestLogger("test")

		fakeMarshaler = &hfakes.Marshaler{}
		fakeMarshaler.MarshalStub = json.Marshal
		fakePolicyCleaner = &fakes.PolicyCleaner{}
		fakeErrorResponse = &fakes.ErrorResponse{}

		handler = &handlers.PoliciesCleanup{
			Logger:        logger,
			Marshaler:     fakeMarshaler,
			PolicyCleaner: fakePolicyCleaner,
			ErrorResponse: fakeErrorResponse,
		}

		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}

		fakePolicyCleaner.DeleteStalePoliciesReturns(policies, nil)
		resp = httptest.NewRecorder()
		request, _ = http.NewRequest("POST", "/networking/v0/external/policies/cleanup", nil)
	})

	It("Cleans up stale policies for deleted apps", func() {
		handler.ServeHTTP(resp, request, tokenData)

		Expect(fakePolicyCleaner.DeleteStalePoliciesCallCount()).To(Equal(1))
		Expect(fakeMarshaler.MarshalCallCount()).To(Equal(1))

		for i, _ := range policies {
			policies[i].Source.Tag = ""
			policies[i].Destination.Tag = ""
		}
		deletedPolicies := struct {
			TotalPolicies int             `json:"total_policies"`
			Policies      []models.Policy `json:"policies"`
		}{1, policies}

		Expect(fakeMarshaler.MarshalArgsForCall(0)).To(Equal(deletedPolicies))

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON(`{
			"total_policies":1,
			"policies": [
			{
				"source": {
					"id": "live-guid"
				},
				"destination": {
					"id": "dead-guid",
					"protocol": "tcp",
					"port": 8080
				}
			}
			]
		}
			`))
	})

	Context("When deleting the policies fails", func() {
		BeforeEach(func() {
			fakePolicyCleaner.DeleteStalePoliciesReturns(nil, errors.New("potato"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("potato"))
			Expect(message).To(Equal("policies-cleanup"))
			Expect(description).To(Equal("policies cleanup failed"))
		})
	})

	Context("When marshalling the reponse fails", func() {
		BeforeEach(func() {
			fakeMarshaler.MarshalReturns(nil, errors.New("potato"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("potato"))
			Expect(message).To(Equal("policies-cleanup"))
			Expect(description).To(Equal("marshal response failed"))
		})
	})
})
