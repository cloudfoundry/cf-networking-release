package handlers_test

import (
	"encoding/json"
	"errors"
	lfakes "lib/fakes"
	"net/http"
	"net/http/httptest"
	"policy-server/fakes"
	"policy-server/handlers"
	"policy-server/models"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PoliciesCleanup", func() {
	var (
		request       *http.Request
		handler       *handlers.PoliciesCleanup
		resp          *httptest.ResponseRecorder
		fakeStore     *fakes.Store
		fakeUAAClient *fakes.UAAClient
		fakeCCClient  *fakes.CCClient
		logger        *lagertest.TestLogger
		fakeMarshaler *lfakes.Marshaler
		allPolicies   []models.Policy
	)

	BeforeEach(func() {
		allPolicies = []models.Policy{{
			Source: models.Source{ID: "live-guid", Tag: "tag"},
			Destination: models.Destination{
				ID:       "live-guid",
				Tag:      "tag",
				Protocol: "tcp",
				Port:     8080,
			},
		}, {
			Source: models.Source{ID: "dead-guid", Tag: "tag"},
			Destination: models.Destination{
				ID:       "live-guid",
				Tag:      "tag",
				Protocol: "udp",
				Port:     1234,
			},
		}, {
			Source: models.Source{ID: "live-guid", Tag: "tag"},
			Destination: models.Destination{
				ID:       "dead-guid",
				Tag:      "tag",
				Protocol: "udp",
				Port:     1234,
			},
		}}

		fakeStore = &fakes.Store{}
		fakeUAAClient = &fakes.UAAClient{}
		fakeCCClient = &fakes.CCClient{}
		logger = lagertest.NewTestLogger("test")

		fakeMarshaler = &lfakes.Marshaler{}
		fakeMarshaler.MarshalStub = json.Marshal
		handler = &handlers.PoliciesCleanup{
			Logger:    logger,
			Store:     fakeStore,
			UAAClient: fakeUAAClient,
			CCClient:  fakeCCClient,
			Marshaler: fakeMarshaler,
		}

		resp = httptest.NewRecorder()
		request, _ = http.NewRequest("POST", "/networking/v0/external/policies/cleanup", nil)

		fakeUAAClient.GetTokenReturns("valid-token", nil)
		fakeStore.AllReturns(allPolicies, nil)
		fakeCCClient.GetAllAppGUIDsReturns(map[string]interface{}{"live-guid": nil}, nil)
	})

	It("Returns the policies which should be cleaned up without tags", func() {

		handler.ServeHTTP(resp, request, "")
		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
		Expect(fakeCCClient.GetAllAppGUIDsCallCount()).To(Equal(1))
		Expect(fakeCCClient.GetAllAppGUIDsArgsForCall(0)).To(Equal("valid-token"))
		Expect(fakeMarshaler.MarshalCallCount()).To(Equal(1))
		policies := allPolicies[1:]
		for i, _ := range policies {
			policies[i].Source.Tag = ""
			policies[i].Destination.Tag = ""
		}
		policyCleanup := struct {
			TotalPolicies int             `json:"total_policies"`
			Policies      []models.Policy `json:"policies"`
		}{2, policies}
		Expect(fakeMarshaler.MarshalArgsForCall(0)).To(Equal(policyCleanup))

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON(`{
			"total_policies":2,
			"policies": [
			{
				"source": {
					"id": "dead-guid"
				},
				"destination": {
					"id": "live-guid",
					"protocol": "udp",
					"port": 1234
				}
			},
			{
				"source": {
					"id": "live-guid"
				},
				"destination": {
					"id": "dead-guid",
					"protocol": "udp",
					"port": 1234
				}
			}
			]
		}
			`))

	})

	Context("When retrieving policies from the db fails", func() {
		BeforeEach(func() {
			fakeStore.AllReturns(nil, errors.New("potato"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database read failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(logger).To(gbytes.Say("store-list-policies-failed.*potato"))
		})
	})

	Context("When getting the UAA token fails", func() {
		BeforeEach(func() {
			fakeUAAClient.GetTokenReturns("", errors.New("potato"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "get UAA token failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(logger).To(gbytes.Say("get-uaa-token-failed.*potato"))
		})
	})

	Context("When getting the apps from the Cloud-Controller fails", func() {
		BeforeEach(func() {
			fakeCCClient.GetAllAppGUIDsReturns(nil, errors.New("potato"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "get app guids from Cloud-Controller failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(logger).To(gbytes.Say("cc-get-app-guids-failed.*potato"))
		})
	})

	Context("When marshalling the reponse fails", func() {
		BeforeEach(func() {
			fakeMarshaler.MarshalReturns(nil, errors.New("potato"))
		})
		It("responds with 500", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "marshal response failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, "")
			Expect(logger).To(gbytes.Say("marshal-failed.*potato"))
		})
	})

})
