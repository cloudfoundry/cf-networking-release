package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/uaa_client"

	apifakes "policy-server/api/fakes"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	"policy-server/store"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesDelete", func() {
	var (
		requestBody       string
		request           *http.Request
		handler           *handlers.PoliciesDelete
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.PolicyStore
		fakeMapper        *apifakes.PolicyMapper
		logger            *lagertest.TestLogger
		expectedLogger    lager.Logger
		expectedPolicies  []store.Policy
		fakePolicyGuard   *fakes.PolicyGuard
		fakeErrorResponse *fakes.ErrorResponse
		tokenData         uaa_client.CheckTokenResponse
	)

	const Route = "/networking/v0/external/policies/delete"

	BeforeEach(func() {
		var err error
		requestBody = "some request body"
		request, err = http.NewRequest("POST", Route, bytes.NewBuffer([]byte(requestBody)))
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.PolicyStore{}
		fakeMapper = &apifakes.PolicyMapper{}
		fakePolicyGuard = &fakes.PolicyGuard{}
		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("delete-policies")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.PoliciesDelete{
			Mapper:        fakeMapper,
			Store:         fakeStore,
			PolicyGuard:   fakePolicyGuard,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()

		expectedPolicies = []store.Policy{
			{
				Source: store.Source{ID: "some-app-guid"},
				Destination: store.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Ports: store.Ports{
						Start: 8080,
						End:   8080,
					},
				},
			},
		}

		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}
		fakeMapper.AsStorePolicyReturns(expectedPolicies, nil)
		fakePolicyGuard.CheckAccessReturns(true, nil)
	})

	It("removes the entry from the policy server", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

		Expect(fakeMapper.AsStorePolicyCallCount()).To(Equal(1))
		Expect(fakeMapper.AsStorePolicyArgsForCall(0)).To(Equal([]byte(requestBody)))

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
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

		Expect(logger.Logs()).To(HaveLen(1))
		Expect(logger.Logs()[0]).To(SatisfyAll(
			LogsWith(lager.INFO, "test.delete-policies.deleted-policies"),
			HaveLogData(SatisfyAll(
				HaveLen(3),
				HaveKeyWithValue("policies", SatisfyAll(
					HaveLen(1),
					ConsistOf(
						SatisfyAll(
							HaveKeyWithValue("Source", HaveKeyWithValue("ID", "some-app-guid")),
							HaveKeyWithValue("Destination", SatisfyAll(
								HaveKeyWithValue("ID", "some-other-app-guid"),
								HaveKeyWithValue("Protocol", "tcp"),
								HaveKeyWithValue("Ports", SatisfyAll(
									HaveLen(2),
									HaveKeyWithValue("Start", BeEquivalentTo(8080)),
									HaveKeyWithValue("End", BeEquivalentTo(8080)),
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

	Context("when the logger isn't on the request context", func() {
		BeforeEach(func() {
			logger = nil
		})
		It("still works", func() {
			MakeRequestWithAuth(handler.ServeHTTP, resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON("{}"))
		})
	})

	Context("when the token isn't on the request context", func() {
		BeforeEach(func() {
			tokenData = uaa_client.CheckTokenResponse{}
		})
		It("still works", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON("{}"))

			_, token := fakePolicyGuard.CheckAccessArgsForCall(0)
			Expect(token).To(Equal(tokenData))
		})
	})

	Context("when the mapper fails to get store policies", func() {
		BeforeEach(func() {
			fakeMapper.AsStorePolicyReturns([]store.Policy{}, errors.New("banana"))
		})
		It("calls the bad request header, and logs the error", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("mapper: banana"))
		})
	})

	Context("when the policy guard returns false", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, nil)
		})

		It("calls the forbidden handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.ForbiddenCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.ForbiddenArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("one or more applications cannot be found or accessed"))
			Expect(description).To(Equal("one or more applications cannot be found or accessed"))
		})
	})

	Context("when the policy guard returns an error", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("check access failed"))
		})
	})

	Context("when reading the request body fails", func() {
		BeforeEach(func() {
			request.Body = &testsupport.BadReader{}
		})

		It("calls the bad request handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("invalid request body"))
		})
	})

	Context("when deleting from the store fails", func() {
		BeforeEach(func() {
			fakeStore.DeleteReturns(errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("database delete failed"))
		})
	})
})
