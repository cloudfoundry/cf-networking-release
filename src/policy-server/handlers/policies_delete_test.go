package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	lfakes "lib/fakes"
	"lib/testsupport"
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

var _ = Describe("PoliciesDelete", func() {
	var (
		requestJSON       string
		request           *http.Request
		handler           *handlers.PoliciesDelete
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.Store
		logger            *lagertest.TestLogger
		fakeUnmarshaler   *lfakes.Unmarshaler
		expectedPolicies  []models.Policy
		fakeValidator     *fakes.Validator
		fakePolicyGuard   *fakes.PolicyGuard
		fakeMetricsSender *fakes.MetricsSender
		tokenData         uaa_client.CheckTokenResponse
	)

	const Route = "/networking/v0/external/policies/delete"

	BeforeEach(func() {
		var err error
		requestJSON = `{"policies": [
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
		request, err = http.NewRequest("POST", Route, bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.Store{}
		fakeValidator = &fakes.Validator{}
		fakePolicyGuard = &fakes.PolicyGuard{}
		logger = lagertest.NewTestLogger("test")
		fakeUnmarshaler = &lfakes.Unmarshaler{}
		fakeUnmarshaler.UnmarshalStub = json.Unmarshal
		fakeMetricsSender = &fakes.MetricsSender{}
		handler = &handlers.PoliciesDelete{
			Logger:        logger,
			Unmarshaler:   fakeUnmarshaler,
			Store:         fakeStore,
			Validator:     fakeValidator,
			PolicyGuard:   fakePolicyGuard,
			MetricsSender: fakeMetricsSender,
		}
		resp = httptest.NewRecorder()

		expectedPolicies = []models.Policy{{
			Source: models.Source{ID: "some-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Port:     8080,
			},
		}}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}
		fakePolicyGuard.CheckAccessReturns(true, nil)
	})

	It("removes the entry from the policy server", func() {
		handler.ServeHTTP(resp, request, tokenData)

		Expect(fakeUnmarshaler.UnmarshalCallCount()).To(Equal(1))
		bodyBytes, _ := fakeUnmarshaler.UnmarshalArgsForCall(0)
		Expect(bodyBytes).To(Equal([]byte(requestJSON)))
		Expect(fakePolicyGuard.CheckAccessCallCount()).To(Equal(1))
		policies, token := fakePolicyGuard.CheckAccessArgsForCall(0)
		Expect(policies).To(Equal(expectedPolicies))
		Expect(token).To(Equal(tokenData))
		Expect(fakeStore.DeleteCallCount()).To(Equal(1))
		Expect(fakeStore.DeleteArgsForCall(0)).To(Equal(expectedPolicies))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON("{}"))
	})

	It("logs the policy with username and app guid", func() {
		handler.ServeHTTP(resp, request, tokenData)
		Expect(logger).To(gbytes.Say("policy-delete.*some-app-guid.*some_user"))
	})

	Context("when the policy guard returns false", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, nil)
		})

		It("responds with code 403", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusForbidden))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "one or more applications cannot be found or accessed"}`))
		})

		It("logs the failure", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("check-access-failed.*one or more applications cannot be found or accessed"))
		})
	})

	Context("when the policy guard returns an error", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, errors.New("banana"))
		})

		It("responds with code 500 and a useful error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "banana"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("check-access-failed.*banana"))
		})

		It("increments the counter", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("ExternalPoliciesDeleteError"))
		})
	})

	Context("when a policy to delete includes any validation error", func() {
		BeforeEach(func() {
			var err error
			requestJSON = `{}`
			request, err = http.NewRequest("POST", Route, bytes.NewBuffer([]byte(requestJSON)))
			Expect(err).NotTo(HaveOccurred())

			fakeValidator.ValidatePoliciesReturns(errors.New("banana"))
		})

		It("responds with code 400 and a useful error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "banana"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("bad-request.*banana"))
		})
	})

	Context("when reading the request body fails", func() {
		BeforeEach(func() {
			request.Body = &testsupport.BadReader{}
		})

		It("returns 400 and logs the error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(Equal(`{"error": "invalid request body format passed to API should be JSON"}`))
			Expect(logger).To(gbytes.Say("body-read-failed"))
		})
	})

	Context("when unmarshaling the json fails", func() {
		BeforeEach(func() {
			fakeUnmarshaler.UnmarshalReturns(errors.New("banana"))
		})
		It("returns 400 and logs the error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(Equal(`{"error": "invalid values passed to API"}`))
			Expect(logger).To(gbytes.Say("unmarshal-failed"))
		})
	})

	Context("when deleting from the store fails", func() {
		BeforeEach(func() {
			fakeStore.DeleteReturns(errors.New("banana"))
		})

		It("returns 500 and logs the error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(Equal(`{"error": "database delete failed"}`))
			Expect(logger).To(gbytes.Say("store-delete-failed"))
		})

		It("increments the counter", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("ExternalPoliciesDeleteError"))
		})
	})
})
