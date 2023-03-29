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

var _ = Describe("Health handler", func() {
	var (
		handler           *handlers.Health
		request           *http.Request
		fakeStore         *storeFakes.Store
		fakeErrorResponse *fakes.ErrorResponse
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
		expectedLogger    lager.Logger
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/health", nil)
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &storeFakes.Store{}
		fakeErrorResponse = &fakes.ErrorResponse{}

		handler = &handlers.Health{
			Store:         fakeStore,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()

		logger = lagertest.NewTestLogger("test-logger")
		expectedLogger = lager.NewLogger("test-logger").Session("health")

		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	It("checks the database is up and returns a 200", func() {
		MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

		Expect(fakeStore.CheckDatabaseCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
	})

	Context("when the logger is not provided", func() {
		It("still works", func() {
			handler.ServeHTTP(resp, request)

			Expect(fakeStore.CheckDatabaseCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusOK))
		})
	})

	Context("when the database returns an error", func() {
		BeforeEach(func() {
			fakeStore.CheckDatabaseReturns(errors.New("pineapple"))
		})

		It("calls the internal server error handler", func() {
			MakeRequestWithLogger(handler.ServeHTTP, resp, request, logger)

			Expect(fakeStore.CheckDatabaseCallCount()).To(Equal(1))
			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			l, w, err, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("pineapple"))
			Expect(description).To(Equal("check database failed"))
		})
	})
})
