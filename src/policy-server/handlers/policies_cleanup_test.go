package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/store"

	apifakes "policy-server/api/fakes"

	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesCleanup", func() {
	var (
		request                    *http.Request
		handler                    *handlers.PoliciesCleanup
		resp                       *httptest.ResponseRecorder
		logger                     *lagertest.TestLogger
		expectedLogger             lager.Logger
		fakePolicyCleaner          *fakes.PolicyCleaner
		fakePolicyCollectionWriter *apifakes.PolicyCollectionWriter
		fakeErrorResponse          *fakes.ErrorResponse
		policies                   []store.Policy
		egressPolicies             []store.EgressPolicy
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
		egressPolicies = []store.EgressPolicy{{
			Source: store.EgressSource{ID: "live-guid", Type: "app"},
			Destination: store.EgressDestination{
				Protocol: "tcp",
				IPRanges: []store.IPRange{
					{
						Start: "1.2.3.4",
						End:   "1.2.3.5",
					},
				},
				Ports: []store.Ports{
					{
						Start: 8080,
						End:   8080,
					},
				},
			},
		}}

		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("cleanup-policies")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		fakePolicyCollectionWriter = &apifakes.PolicyCollectionWriter{}
		fakePolicyCleaner = &fakes.PolicyCleaner{}
		fakeErrorResponse = &fakes.ErrorResponse{}

		handler = &handlers.PoliciesCleanup{
			PolicyCollectionWriter: fakePolicyCollectionWriter,
			PolicyCleaner:          fakePolicyCleaner,
			ErrorResponse:          fakeErrorResponse,
		}

		fakePolicyCleaner.DeleteStalePoliciesReturns(policies, egressPolicies, nil)
		fakePolicyCollectionWriter.AsBytesReturns([]byte("some-bytes"), nil)
		resp = httptest.NewRecorder()
		request, _ = http.NewRequest("POST", "/networking/v0/external/policies/cleanup", nil)
	})

	It("Cleans up stale policies for deleted apps", func() {
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakePolicyCleaner.DeleteStalePoliciesCallCount()).To(Equal(1))
		Expect(fakePolicyCollectionWriter.AsBytesCallCount()).To(Equal(1))

		policiesArg, egressPoliciesArg := fakePolicyCollectionWriter.AsBytesArgsForCall(0)

		Expect(policiesArg).To(Equal(policies))
		Expect(egressPoliciesArg).To(Equal(egressPolicies))

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
			fakePolicyCleaner.DeleteStalePoliciesReturns(policies, egressPolicies, errors.New("potato"))
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
			fakePolicyCollectionWriter.AsBytesReturns(nil, errors.New("potato"))
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
