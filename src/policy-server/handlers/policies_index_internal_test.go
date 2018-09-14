package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	storeFakes "policy-server/store/fakes"

	apifakes "policy-server/api/fakes"

	"code.cloudfoundry.org/lager"

	"policy-server/store"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesIndexInternal", func() {
	var (
		handler                    *handlers.PoliciesIndexInternal
		resp                       *httptest.ResponseRecorder
		fakeStore                  *storeFakes.Store
		fakeEgressStore            *fakes.EgressPolicyStore
		fakeErrorResponse          *fakes.ErrorResponse
		logger                     *lagertest.TestLogger
		expectedLogger             lager.Logger
		fakePolicyCollectionWriter *apifakes.PolicyCollectionWriter
		expectedResponseBody       []byte
	)

	BeforeEach(func() {
		allPolicies := []store.Policy{{
			Source: store.Source{ID: "some-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}, {
			Source: store.Source{ID: "another-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 1234,
					End:   1234,
				},
			},
		},
		}

		byGuidsPolicies := []store.Policy{{
			Source: store.Source{ID: "some-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}}

		allEgressPolicies := []store.EgressPolicy{{
			Source: store.EgressSource{ID: "some-egress-app-guid"},
			Destination: store.EgressDestination{
				Protocol: "tcp",
				IPRanges: []store.IPRange{{Start: "8.0.8.0", End: "8.0.8.0"}},
			},
		}}

		expectedResponseBody = []byte("some-response")

		fakePolicyCollectionWriter = &apifakes.PolicyCollectionWriter{}
		fakeStore = &storeFakes.Store{}
		fakeStore.AllReturns(allPolicies, nil)
		fakeEgressStore = &fakes.EgressPolicyStore{}
		fakeEgressStore.ByGuidsReturns(allEgressPolicies, nil)
		fakeStore.ByGuidsReturns(byGuidsPolicies, nil)
		fakePolicyCollectionWriter.AsBytesReturns(expectedResponseBody, nil)
		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("index-policies-internal")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.PoliciesIndexInternal{
			Logger:                 logger,
			Store:                  fakeStore,
			EgressStore:            fakeEgressStore,
			PolicyCollectionWriter: fakePolicyCollectionWriter,
			ErrorResponse:          fakeErrorResponse,
		}
		resp = httptest.NewRecorder()
	})

	It("returns the policies returned by ByGuids", func() {
		request, err := http.NewRequest("GET", "/networking/v0/internal/policies?id=some-app-guid", nil)
		Expect(err).NotTo(HaveOccurred())
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakeStore.ByGuidsCallCount()).To(Equal(1))
		Expect(fakeEgressStore.ByGuidsCallCount()).To(Equal(1))
		srcGuids, dstGuids, inSourceAndDest := fakeStore.ByGuidsArgsForCall(0)
		Expect(srcGuids).To(Equal([]string{"some-app-guid"}))
		Expect(dstGuids).To(Equal([]string{"some-app-guid"}))
		Expect(inSourceAndDest).To(BeFalse())
		guids := fakeEgressStore.ByGuidsArgsForCall(0)
		Expect(guids).To(Equal([]string{"some-app-guid"}))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	})

	Context("when the logger isn't on the request context", func() {
		It("still works", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies?id=some-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
		})
	})

	Context("when there are policies and no ids are passed", func() {
		It("returns all of them", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(fakeEgressStore.AllCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
		})
	})

	Context("when rendering the policies as bytes fails", func() {
		BeforeEach(func() {
			fakePolicyCollectionWriter.AsBytesReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("map policies as bytes failed"))
		})
	})

	Context("when store.All() throws an error", func() {

		BeforeEach(func() {
			fakeStore.AllReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	Context("when store.ByGuids() throws an error", func() {

		BeforeEach(func() {
			fakeStore.ByGuidsReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies?id=some-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("database read failed"))
		})
	})

	Context("when egressStore.All() throws an error", func() {

		BeforeEach(func() {
			fakeEgressStore.AllReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("egress database read failed"))
		})

	})

	Context("when egressStore.ByGuids() throws an error", func() {

		BeforeEach(func() {
			fakeEgressStore.ByGuidsReturns(nil, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies?id=meowmeow", nil)
			Expect(err).NotTo(HaveOccurred())
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("banana"))
			Expect(description).To(Equal("egress database read failed"))
		})

	})
})
