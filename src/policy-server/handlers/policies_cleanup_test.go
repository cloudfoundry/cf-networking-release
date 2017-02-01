package handlers_test

import (
	"encoding/json"
	"errors"
	lfakes "lib/fakes"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PoliciesCleanup", func() {
	var (
		request           *http.Request
		handler           *handlers.PoliciesCleanup
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
		fakePolicyCleaner *fakes.PolicyCleaner
		fakeMarshaler     *lfakes.Marshaler
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

		fakeMarshaler = &lfakes.Marshaler{}
		fakeMarshaler.MarshalStub = json.Marshal
		fakePolicyCleaner = &fakes.PolicyCleaner{}

		handler = &handlers.PoliciesCleanup{
			Logger:        logger,
			Marshaler:     fakeMarshaler,
			PolicyCleaner: fakePolicyCleaner,
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

		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "policies cleanup failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("policies-cleanup.*potato"))
		})
	})

	Context("When marshalling the reponse fails", func() {
		BeforeEach(func() {
			fakeMarshaler.MarshalReturns(nil, errors.New("potato"))
		})

		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "marshal response failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("marshal-failed.*potato"))
		})
	})
})
