package handlers_test

import (
	"bytes"
	"errors"
	"io/ioutil"
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

var _ = Describe("PoliciesCreate", func() {
	var (
		requestBody            string
		request                *http.Request
		handler                *handlers.PoliciesCreate
		resp                   *httptest.ResponseRecorder
		expectedPolicies       []store.Policy
		fakeStore              *fakes.PolicyStore
		fakeMapper             *apifakes.PolicyMapper
		fakePolicyGuard        *fakes.PolicyGuard
		fakeQuotaGuard         *fakes.QuotaGuard
		fakeErrorResponse      *fakes.ErrorResponse
		logger                 *lagertest.TestLogger
		expectedLogger         lager.Logger
		tokenData              uaa_client.CheckTokenResponse
		createPoliciesSucceeds func()
	)

	BeforeEach(func() {
		var err error
		requestBody = "some request body"
		request, err = http.NewRequest("POST", "/networking/v0/external/policies", bytes.NewBuffer([]byte(requestBody)))
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.PolicyStore{}
		fakeMapper = &apifakes.PolicyMapper{}
		fakePolicyGuard = &fakes.PolicyGuard{}
		fakeQuotaGuard = &fakes.QuotaGuard{}
		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("create-policies")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.PoliciesCreate{
			Store:         fakeStore,
			Mapper:        fakeMapper,
			PolicyGuard:   fakePolicyGuard,
			QuotaGuard:    fakeQuotaGuard,
			ErrorResponse: fakeErrorResponse,
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}

		expectedPolicies = []store.Policy{
			{
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
			},
		}

		fakeMapper.AsStorePolicyReturns(expectedPolicies, nil)
		fakePolicyGuard.CheckAccessReturns(true, nil)
		fakeQuotaGuard.CheckAccessReturns(true, nil)
		resp = httptest.NewRecorder()

		createPoliciesSucceeds = func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeMapper.AsStorePolicyCallCount()).To(Equal(1))
			Expect(fakeMapper.AsStorePolicyArgsForCall(0)).To(Equal([]byte(requestBody)))

			Expect(fakePolicyGuard.CheckAccessCallCount()).To(Equal(1))
			policies, token := fakePolicyGuard.CheckAccessArgsForCall(0)
			Expect(policies).To(Equal(expectedPolicies))
			Expect(token).To(Equal(tokenData))
			Expect(fakeStore.CreateCallCount()).To(Equal(1))
			Expect(fakeStore.CreateArgsForCall(0)).To(Equal(expectedPolicies))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON("{}"))
		}
	})
	It("persists a new policy rule", func() {
		createPoliciesSucceeds()
	})

	It("logs the policy with username and app guid", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

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
							HaveKeyWithValue("Source", HaveKeyWithValue("ID", "some-app-guid")),
							HaveKeyWithValue("Destination", SatisfyAll(
								HaveKeyWithValue("ID", "some-other-app-guid"),
								HaveKeyWithValue("Protocol", "tcp"),
								HaveKeyWithValue("Ports", SatisfyAll(
									HaveLen(2),
									HaveKeyWithValue("Start", BeEquivalentTo(8080)),
									HaveKeyWithValue("End", BeEquivalentTo(9090)),
								)),
							)),
						),
						SatisfyAll(
							HaveKeyWithValue("Source", HaveKeyWithValue("ID", "another-app-guid")),
							HaveKeyWithValue("Destination", SatisfyAll(
								HaveKeyWithValue("ID", "some-other-app-guid"),
								HaveKeyWithValue("Protocol", "udp"),
								HaveKeyWithValue("Ports", SatisfyAll(
									HaveLen(2),
									HaveKeyWithValue("Start", BeEquivalentTo(1234)),
									HaveKeyWithValue("End", BeEquivalentTo(1234)),
								)),
							)),
						),
					),
				)),
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
			Expect(resp.Body.Bytes()).To(MatchJSON("{}"))
		})
	})

	Context("when the token isn't on the request context", func() {
		BeforeEach(func() {
			tokenData = uaa_client.CheckTokenResponse{}
		})
		It("still works", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(MatchJSON("{}"))
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

	Context("when the quota guard returns false", func() {
		BeforeEach(func() {
			fakeQuotaGuard.CheckAccessReturns(false, nil)
		})

		It("calls the forbidden handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.ForbiddenCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.ForbiddenArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("policy quota exceeded"))
			Expect(description).To(Equal("policy quota exceeded"))
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

	Context("when the quota guard returns an error", func() {
		BeforeEach(func() {
			fakeQuotaGuard.CheckAccessReturns(false, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("check quota failed"))
		})
	})

	Context("when the store Create call returns an error", func() {
		BeforeEach(func() {
			fakeStore.CreateReturns(errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("database create failed"))
		})
	})

	Context("when there are errors reading the body bytes", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(&testsupport.BadReader{})
		})

		It("calls the bad request handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, tokenData)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("failed reading request body"))
		})
	})
})
