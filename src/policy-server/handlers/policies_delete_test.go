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
		fakeErrorResponse *fakes.ErrorResponse
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
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.PoliciesDelete{
			Logger:        logger,
			Unmarshaler:   fakeUnmarshaler,
			Store:         fakeStore,
			Validator:     fakeValidator,
			PolicyGuard:   fakePolicyGuard,
			ErrorResponse: fakeErrorResponse,
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

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-delete"))
			Expect(description).To(Equal("check access failed"))
		})
	})

	Context("when a policy to delete includes any validation error", func() {
		BeforeEach(func() {
			fakeValidator.ValidatePoliciesReturns(errors.New("banana"))
		})

		It("calls the bad request handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-delete"))
			Expect(description).To(Equal("banana"))
		})
	})

	Context("when reading the request body fails", func() {
		BeforeEach(func() {
			request.Body = &testsupport.BadReader{}
		})

		It("calls the bad request handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-delete"))
			Expect(description).To(Equal("invalid request body"))
		})
	})

	Context("when unmarshaling the json fails", func() {
		BeforeEach(func() {
			fakeUnmarshaler.UnmarshalReturns(errors.New("banana"))
		})

		It("calls the bad request handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-delete"))
			Expect(description).To(Equal("invalid values passed to API"))
		})
	})

	Context("when deleting from the store fails", func() {
		BeforeEach(func() {
			fakeStore.DeleteReturns(errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-delete"))
			Expect(description).To(Equal("database delete failed"))
		})
	})
})
