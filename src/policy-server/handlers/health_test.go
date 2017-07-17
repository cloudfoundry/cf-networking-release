package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health handler", func() {
	var (
		handler           *handlers.Health
		request           *http.Request
		fakeStore         *fakes.DataStore
		fakeErrorResponse *fakes.ErrorResponse
		resp              *httptest.ResponseRecorder
		logger            *lagertest.TestLogger
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/health", nil)
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.DataStore{}
		fakeErrorResponse = &fakes.ErrorResponse{}

		handler = &handlers.Health{
			Store:         fakeStore,
			ErrorResponse: fakeErrorResponse,
		}
		resp = httptest.NewRecorder()

		logger = lagertest.NewTestLogger("test-logger")
	})

	It("checks the database is up and returns a 200", func() {
		handler.ServeHTTP(logger, resp, request)
		Expect(fakeStore.CheckDatabaseCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusOK))
	})

	Context("when the database returns an error", func() {
		BeforeEach(func() {
			fakeStore.CheckDatabaseReturns(errors.New("pineapple"))
		})

		It("calls the internal server error handler", func() {
			handler.ServeHTTP(logger, resp, request)
			Expect(fakeStore.CheckDatabaseCallCount()).To(Equal(1))
			Expect(fakeErrorResponse.InternalServerErrorCallCount()).To(Equal(1))

			w, err, message, description := fakeErrorResponse.InternalServerErrorArgsForCall(0)
			Expect(w).To(Equal(resp))
			Expect(err).To(MatchError("pineapple"))
			Expect(message).To(Equal("health"))
			Expect(description).To(Equal("check database failed"))

			By("logging the error")
			Expect(logger.Logs()).To(HaveLen(1))
			Expect(logger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test-logger.health.failed-checking-database"),
				HaveLogData(SatisfyAll(
					HaveLen(2),
					HaveKeyWithValue("error", "pineapple"),
					HaveKeyWithValue("session", "1"),
				)),
			))
		})
	})
})
