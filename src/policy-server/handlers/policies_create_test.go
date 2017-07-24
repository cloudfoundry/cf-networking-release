package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/api"
	"policy-server/uaa_client"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"policy-server/store"
)

var _ = FDescribe("PoliciesCreate", func() {
	var (
		requestJSON       string
		request           *http.Request
		handler           *handlers.PoliciesCreate
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.DataStore
		fakeValidator     *fakes.Validator
		fakePolicyGuard   *fakes.PolicyGuard
		fakeQuotaGuard    *fakes.QuotaGuard
		fakeErrorResponse *fakes.ErrorResponse
		logger            *lagertest.TestLogger
		fakeUnmarshaler   *hfakes.Unmarshaler
		tokenData         uaa_client.CheckTokenResponse
	)

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
					"ports": {
						"start": 8080,
						"end": 9090
					}
				}
			},
			{
				"source": {
					"id": "another-app-guid"
				},
				"destination": {
					"id": "some-other-app-guid",
					"protocol": "udp",
					"ports": {
						"start": 1234,
						"end": 1234
					}
				}
			}
        ]}`
		request, err = http.NewRequest("POST", "/networking/v0/external/policies", bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())
		request.Header["Accept"] = []string{"1.0.0+policy_server-json" }

		fakeStore = &fakes.DataStore{}
		fakeValidator = &fakes.Validator{}
		fakePolicyGuard = &fakes.PolicyGuard{}
		fakeQuotaGuard = &fakes.QuotaGuard{}
		logger = lagertest.NewTestLogger("test")
		fakeUnmarshaler = &hfakes.Unmarshaler{}
		fakeErrorResponse = &fakes.ErrorResponse{}
		fakeUnmarshaler.UnmarshalStub = json.Unmarshal
		handler = &handlers.PoliciesCreate{
			Store:         fakeStore,
			Unmarshaler:   fakeUnmarshaler,
			Validator:     fakeValidator,
			PolicyGuard:   fakePolicyGuard,
			QuotaGuard:    fakeQuotaGuard,
			ErrorResponse: fakeErrorResponse,
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}
		fakePolicyGuard.CheckAccessReturns(true, nil)
		fakeQuotaGuard.CheckAccessReturns(true, nil)
		resp = httptest.NewRecorder()
	})
	It("persists a new policy rule", func() {
		expectedPolicies := []api.Policy{{
			Source: api.Source{ID: "some-app-guid"},
			Destination: api.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: api.Ports{
					Start: 8080,
					End:   9090,
				},
			},
		}, {
			Source: api.Source{ID: "another-app-guid"},
			Destination: api.Destination{
				ID:       "some-other-app-guid",
				Protocol: "udp",
				Ports: api.Ports{
					Start: 1234,
					End:   1234,
				},
			},
		}}

		expectedStorePolicies := []store.Policy{{
			Source: store.Source{ID: "some-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   9090,
				},
			},
		}, {
			Source: store.Source{ID: "another-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "udp",
				Ports: store.Ports{
					Start: 1234,
					End:   1234,
				},
			},
		}}

		handler.ServeHTTP(logger, resp, request, tokenData)

		Expect(fakeUnmarshaler.UnmarshalCallCount()).To(Equal(1))
		bodyBytes, _ := fakeUnmarshaler.UnmarshalArgsForCall(0)
		Expect(bodyBytes).To(Equal([]byte(requestJSON)))
		Expect(fakeValidator.ValidatePoliciesCallCount()).To(Equal(1))
		Expect(fakeValidator.ValidatePoliciesArgsForCall(0)).To(Equal(expectedPolicies))
		Expect(fakePolicyGuard.CheckAccessCallCount()).To(Equal(1))
		policies, token := fakePolicyGuard.CheckAccessArgsForCall(0)
		Expect(policies).To(Equal(expectedPolicies))
		Expect(token).To(Equal(tokenData))
		Expect(fakeStore.CreateCallCount()).To(Equal(1))
		Expect(fakeStore.CreateArgsForCall(0)).To(Equal(expectedStorePolicies))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON("{}"))
	})

	It("logs the policy with username and app guid", func() {
		handler.ServeHTTP(logger, resp, request, tokenData)

		By("logging the success")
		Expect(logger.Logs()).To(HaveLen(1))
		Expect(logger.Logs()[0]).To(SatisfyAll(
			LogsWith(lager.INFO, "test.create-policies.created-policies"),
			HaveLogData(SatisfyAll(
				HaveLen(3),
				HaveKeyWithValue("policies", SatisfyAll(
					HaveLen(2),
					ConsistOf(
						SatisfyAll(
							HaveKeyWithValue("source", HaveKeyWithValue("id", "some-app-guid")),
							HaveKeyWithValue("destination", SatisfyAll(
								HaveLen(3),
								HaveKeyWithValue("id", "some-other-app-guid"),
								HaveKeyWithValue("protocol", "tcp"),
								HaveKeyWithValue("ports", SatisfyAll(
									HaveLen(2),
									HaveKeyWithValue("start", BeEquivalentTo(8080)),
									HaveKeyWithValue("end", BeEquivalentTo(9090)),
								)),
							)),
						),
						SatisfyAll(
							HaveKeyWithValue("source", HaveKeyWithValue("id", "another-app-guid")),
							HaveKeyWithValue("destination", SatisfyAll(
								HaveLen(3),
								HaveKeyWithValue("id", "some-other-app-guid"),
								HaveKeyWithValue("protocol", "udp"),
								HaveKeyWithValue("ports", SatisfyAll(
									HaveLen(2),
									HaveKeyWithValue("start", BeEquivalentTo(1234)),
									HaveKeyWithValue("end", BeEquivalentTo(1234)),
								)),
							)),
						),
					),
				)),
				HaveKeyWithValue("session", "1"),
				HaveKeyWithValue("userName", "some_user"),
			)),
		))

	})

	Context("when the policy guard returns false", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, nil)
		})

		It("calls the forbidden handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.ForbiddenCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.ForbiddenArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("one or more applications cannot be found or accessed"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("one or more applications cannot be found or accessed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.failed-authorizing"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "one or more applications cannot be found or accessed"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("when the quota guard returns false", func() {
		BeforeEach(func() {
			fakeQuotaGuard.CheckAccessReturns(false, nil)
		})

		It("calls the forbidden handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.ForbiddenCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.ForbiddenArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("policy quota exceeded"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("policy quota exceeded"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.quota-exceeded"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "policy quota exceeded"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("when the validator fails", func() {
		BeforeEach(func() {
			fakeValidator.ValidatePoliciesReturns(errors.New("banana"))
		})

		It("calls the bad request handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("banana"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.failed-validating-policies"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("when the policy guard returns an error", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("check access failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.failed-checking-access"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("when the quota guard returns an error", func() {
		BeforeEach(func() {
			fakeQuotaGuard.CheckAccessReturns(false, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("check quota failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.failed-checking-quota"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("when the store Create call returns an error", func() {
		BeforeEach(func() {
			fakeStore.CreateReturns(errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("database create failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.failed-creating-in-database"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))

		})
	})

	Context("when there are errors reading the body bytes", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(&testsupport.BadReader{})
		})

		It("calls the bad request handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("failed reading request body"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.failed-reading-request-body"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))

		})
	})

	Context("when there are errors in the request body formatting", func() {
		BeforeEach(func() {
			fakeUnmarshaler.UnmarshalReturns(errors.New("banana"))
		})

		It("calls the bad request handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("invalid values passed to API"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.create-policies.failed-unmarshalling-payload"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "banana"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})

	Context("when the api header version is not compatible", func() {
		BeforeEach(func() {
			request.Header["Accept"] = []string{"0.0.0+policy_server-toml" }
		})

		It("calls the 406 Not Acceptable handler", func() {
			handler.ServeHTTP(logger, resp, request, tokenData)

			Expect(fakeErrorResponse.NotAcceptableCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.NotAcceptableArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("version-mismatch"))
			Expect(message).To(Equal("policies-create"))
			Expect(description).To(Equal("invalid Accept Header passed to API"))

			By("logging")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.INFO, "test.create-policies.failed-validating-version"),
			))
		})

		Context("when multiple accept values are provided", func() {
			It("should return a sensible error", func() {
				Fail("implement me")
			})
		})
	})
})
