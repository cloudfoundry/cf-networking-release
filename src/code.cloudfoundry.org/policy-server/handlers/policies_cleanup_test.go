package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	apifakes "code.cloudfoundry.org/policy-server/api/fakes"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/handlers/fakes"
	"code.cloudfoundry.org/policy-server/store"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesCleanup", func() {
	var (
		request           *http.Request
		handler           *handlers.PoliciesCleanup
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
		expectedLogger    lager.Logger
		fakePolicyCleaner *fakes.PolicyCleaner
		fakePolicyMapper  *apifakes.PolicyMapper
		fakeErrorResponse *fakes.ErrorResponse
		policies          []store.Policy
	)

	BeforeEach(func() {
		policies = []store.Policy{{
			Source: store.Source{ID: "live-guid", Tag: "tag"},
			Destination: store.Destination{
				ID:       "dead-guid",
				Tag:      "tag",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}}

		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("cleanup-policies")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		fakePolicyMapper = &apifakes.PolicyMapper{}
		fakePolicyCleaner = &fakes.PolicyCleaner{}
		fakeErrorResponse = &fakes.ErrorResponse{}

		handler = &handlers.PoliciesCleanup{
			PolicyCleaner: fakePolicyCleaner,
			PolicyMapper:  fakePolicyMapper,
			ErrorResponse: fakeErrorResponse,
		}

		fakePolicyCleaner.DeleteStalePoliciesReturns(policies, nil)
		fakePolicyMapper.AsBytesReturns([]byte("some-bytes"), nil)
		resp = httptest.NewRecorder()
		request, _ = http.NewRequest("POST", "/networking/v0/external/policies/cleanup", nil)
	})

	It("Cleans up stale policies for deleted apps", func() {
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakePolicyCleaner.DeleteStalePoliciesCallCount()).To(Equal(1))
		Expect(fakePolicyMapper.AsBytesCallCount()).To(Equal(1))

		policiesArg := fakePolicyMapper.AsBytesArgsForCall(0)

		Expect(policiesArg).To(Equal(policies))

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(Equal(`some-bytes`))
	})

	Context("when the logger isn't on the request context", func() {
		It("returns all the policies, but does not include the tags", func() {
			handler.ServeHTTP(resp, request)
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(Equal(`some-bytes`))
		})
	})

	Context("When deleting the policies fails", func() {
		BeforeEach(func() {
			fakePolicyCleaner.DeleteStalePoliciesReturns(policies, errors.New("potato"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("potato"))
			Expect(description).To(Equal("policies cleanup failed"))
		})
	})

	Context("When mapping the policies to bytes", func() {
		BeforeEach(func() {
			fakePolicyMapper.AsBytesReturns(nil, errors.New("potato"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("potato"))
			Expect(description).To(Equal("map policy as bytes failed"))
		})
	})
})
