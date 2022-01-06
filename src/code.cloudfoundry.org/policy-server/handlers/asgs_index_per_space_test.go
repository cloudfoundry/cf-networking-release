package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/handlers/fakes"
	"code.cloudfoundry.org/policy-server/store"
	storeFakes "code.cloudfoundry.org/policy-server/store/fakes"
	"code.cloudfoundry.org/policy-server/uaa_client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Asgs per space index handler", func() {
	var (
		bySpaceGuidsAsgs     []store.SecurityGroup
		expectedResponseBody []byte
		// filteredPolicies     []store.Policy
		request   *http.Request
		handler   *handlers.AsgsPerSpaceIndex
		resp      *httptest.ResponseRecorder
		fakeStore *storeFakes.SecurityGroupsStore
		// fakePolicyFilter  *fakes.PolicyFilter
		fakeErrorResponse *fakes.ErrorResponse
		fakeMarshaler     *hfakes.Marshaler
		// fakePolicyGuard   *fakes.PolicyGuard
		logger         *lagertest.TestLogger
		expectedLogger lager.Logger
		token          uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		expectedResponseBody = []byte("some-response")
		bySpaceGuidsAsgs = []store.SecurityGroup{}

		var err error
		request, err = http.NewRequest("GET", "/networking/v1/external/security_group_rules", nil)
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &storeFakes.SecurityGroupsStore{}
		fakeStore.BySpaceGuidsReturns(bySpaceGuidsAsgs, nil)

		// fakePolicyGuard = &fakes.PolicyGuard{}
		// fakePolicyGuard.IsNetworkAdminReturns(true)

		fakeErrorResponse = &fakes.ErrorResponse{}
		fakeMarshaler = &hfakes.Marshaler{}
		// fakePolicyFilter = &fakes.PolicyFilter{}
		// fakePolicyFilter.FilterPoliciesStub = func(policies []store.Policy, subjectToken uaa_client.CheckTokenResponse) ([]store.Policy, error) {
		// 	return filteredPolicies, nil
		// }
		fakeMarshaler.MarshalReturns(expectedResponseBody, nil)
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.AsgsPerSpaceIndex{
			Store:     fakeStore,
			Marshaler: fakeMarshaler,
			// PolicyFilter:  fakePolicyFilter,
			// PolicyGuard:   fakePolicyGuard,
			ErrorResponse: fakeErrorResponse,
		}

		token = uaa_client.CheckTokenResponse{
			Scope: []string{"some-scope", "some-other-scope"},
		}
		resp = httptest.NewRecorder()

		expectedLogger = lager.NewLogger("test").Session("index-security-group-rules-per-space")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	It("returns all the asgs", func() {
		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

		Expect(fakeStore.BySpaceGuidsCallCount()).To(Equal(1))
		// Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	})

	Context("when the logger isn't on the request context", func() {
		It("still works", func() {
			MakeRequestWithAuth(handler.ServeHTTP, resp, request, token)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
		})
	})

	FContext("when from and limit parameters are passed in", func() {
		BeforeEach(func() {
			var err error
			request, err = http.NewRequest("GET", "/networking/v1/external/security_group_rules?from=51&limit=50", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("queries store with Page argument", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeStore.BySpaceGuidsCallCount()).To(Equal(1))
			_, page := fakeStore.BySpaceGuidsArgsForCall(0)
			Expect(page.Limit).To(Equal(50))
			Expect(page.From).To(Equal(51))
		})
	})

	// Context("when the token isn't on the request context", func() {
	// 	It("still works", func() {
	// 		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

	// 		Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
	// 		_, filterToken := fakePolicyFilter.FilterPoliciesArgsForCall(0)
	// 		Expect(filterToken).To(Equal(uaa_client.CheckTokenResponse{}))
	// 		Expect(resp.Code).To(Equal(http.StatusOK))
	// 		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	// 	})
	// })

	Context("when marshalling the asgs as bytes fails", func() {
		BeforeEach(func() {
			fakeMarshaler.MarshalReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("map asgs as bytes failed"))
		})
	})

	// Context("when a list of space guids is provided as a query parameter", func() {
	// 	BeforeEach(func() {
	// 		var err error
	// 		request, err = http.NewRequest("GET", "/networking/v0/external/policies?id=some-app-guid,yet-another-app-guid", nil)
	// 		Expect(err).NotTo(HaveOccurred())
	// 	})

	// 	It("filters on only those policies returned by ByGuids", func() {
	// 		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

	// 		Expect(fakeStore.ByGuidsCallCount()).To(Equal(1))
	// 		srcGuids, destGuids, inSourceAndDest := fakeStore.ByGuidsArgsForCall(0)
	// 		Expect(srcGuids).To(ConsistOf([]string{"some-app-guid", "yet-another-app-guid"}))
	// 		Expect(destGuids).To(ConsistOf([]string{"some-app-guid", "yet-another-app-guid"}))
	// 		Expect(inSourceAndDest).To(BeFalse())
	// 		Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
	// 		policies, subjectToken := fakePolicyFilter.FilterPoliciesArgsForCall(0)
	// 		Expect(policies).To(Equal(byGuidsAPIPolicies))
	// 		Expect(subjectToken).To(Equal(token))
	// 		Expect(resp.Code).To(Equal(http.StatusOK))
	// 	})

	// 	Context("when the id list is empty", func() {
	// 		It("filters on only those policies returned by ByGuids", func() {
	// 			var err error
	// 			request, err = http.NewRequest("GET", "/networking/v0/external/policies?id=", nil)
	// 			Expect(err).NotTo(HaveOccurred())

	// 			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)
	// 			Expect(fakeStore.ByGuidsCallCount()).To(Equal(1))
	// 			srcGuids, destGuids, inSourceAndDest := fakeStore.ByGuidsArgsForCall(0)
	// 			Expect(srcGuids).To(Equal([]string{""}))
	// 			Expect(destGuids).To(Equal([]string{""}))
	// 			Expect(inSourceAndDest).To(BeFalse())
	// 			Expect(fakePolicyFilter.FilterPoliciesCallCount()).To(Equal(1))
	// 			policies, subjectToken := fakePolicyFilter.FilterPoliciesArgsForCall(0)
	// 			Expect(policies).To(Equal(byGuidsAPIPolicies))
	// 			Expect(subjectToken).To(Equal(token))

	// 			Expect(resp.Code).To(Equal(http.StatusOK))
	// 		})
	// 	})
	// })

	Context("when the store throws an error", func() {
		BeforeEach(func() {
			fakeStore.AllReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	// Context("when the policy filter throws an error", func() {
	// 	BeforeEach(func() {
	// 		fakePolicyFilter.FilterPoliciesReturns(nil, errors.New("banana"))
	// 	})

	// 	It("calls the internal server error handler", func() {
	// 		MakeRequestWithLoggerAndAuth(handler.ServeHTTP, resp, request, logger, token)

	// 		Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

	// 		l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
	// 		Expect(l).To(Equal(expectedLogger))
	// 		Expect(w).To(Equal(resp))
	// 		Expect(err).To(MatchError("banana"))
	// 		Expect(description).To(Equal("filter policies failed"))
	// 	})
	// })
})
