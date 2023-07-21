package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/handlers/fakes"
	storeFakes "code.cloudfoundry.org/policy-server/store/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesLastUpdatedInternal", func() {
	var (
		handler              *handlers.PoliciesLastUpdatedInternal
		resp                 *httptest.ResponseRecorder
		fakeStore            *storeFakes.Store
		fakeErrorResponse    *fakes.ErrorResponse
		logger               *lagertest.TestLogger
		expectedLogger       lager.Logger
		expectedResponseBody []byte
	)

	BeforeEach(func() {
		expectedResponseBody = []byte("12345")

		fakeStore = &storeFakes.Store{}
		fakeStore.LastUpdatedReturns(12345, nil)
		logger = lagertest.NewTestLogger("test")
		expectedLogger = lager.NewLogger("test").Session("policies-last-updated-internal")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		fakeErrorResponse = &fakes.ErrorResponse{}
		handler = &handlers.PoliciesLastUpdatedInternal{
			Logger:        logger,
			Store:         fakeStore,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()
	})

	It("returns the last updated returned by LastUpdated", func() {
		request, err := http.NewRequest("GET", "/networking/v0/internal/policies_last_updated", nil)
		Expect(err).NotTo(HaveOccurred())
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakeStore.LastUpdatedCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
	})

	Context("when the logger isn't on the request context", func() {
		It("still works", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies_last_updated", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.Bytes()).To(Equal(expectedResponseBody))
		})
	})

	Context("when store throws an error", func() {
		BeforeEach(func() {
			fakeStore.LastUpdatedReturns(0, errors.New("banana"))
		})

		It("calls the internal server error handler", func() {
			request, err := http.NewRequest("GET", "/networking/v0/internal/policies_last_updated", nil)
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
})
