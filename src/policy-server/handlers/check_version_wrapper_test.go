package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckVersionWrapper", func() {
	var (
		checkVersionHandler http.Handler
		checkVersionWrapper *handlers.CheckVersionWrapper
		fakeHandlerv0       http.Handler
		fakeHandlerv1       http.Handler
		handlerMap          map[string]http.Handler
		logger              *lagertest.TestLogger
		expectedLogger      lager.Logger
		fakeErrorResponse   *fakes.ErrorResponse
		fakeRataAdapter     *fakes.RataAdapter
		request             *http.Request
		resp                *httptest.ResponseRecorder

		v0Count int
		v1Count int
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		expectedLogger = lager.NewLogger("test").Session("check-version")
		testSink := lagertest.NewTestSink()
		expectedLogger.RegisterSink(testSink)
		expectedLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		resp = httptest.NewRecorder()
		request, _ = http.NewRequest("POST", "/some/vwhatever/versioned/resource", nil)

		fakeHandlerv0 = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			v0Count++
			logger.Info("v0")
			w.Write([]byte("v0"))
		})

		fakeHandlerv1 = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			v1Count++
			logger.Info("v1")
			w.Write([]byte("v1"))
		})

		handlerMap = map[string]http.Handler{
			"v0": fakeHandlerv0,
			"v1": fakeHandlerv1,
		}

		fakeErrorResponse = &fakes.ErrorResponse{}
		fakeRataAdapter = &fakes.RataAdapter{}

		checkVersionWrapper = &handlers.CheckVersionWrapper{
			ErrorResponse: fakeErrorResponse,
			RataAdapter:   fakeRataAdapter,
		}
		checkVersionHandler = checkVersionWrapper.CheckVersion(handlerMap)
	})

	AfterEach(func() {
		v0Count = 0
		v1Count = 0
	})

	Context("when the request has a supported version", func() {
		BeforeEach(func() {
			fakeRataAdapter.ParamReturns("v1")
		})
		It("should delegate to handler of the requested version", func() {
			MakeRequestWithLogger(checkVersionHandler.ServeHTTP, resp, request, logger)

			Expect(v0Count).To(Equal(0))
			Expect(v1Count).To(Equal(1))
			Expect(fakeRataAdapter.ParamCallCount()).To(Equal(1))
			req, paramName := fakeRataAdapter.ParamArgsForCall(0)
			Expect(req.WithContext(context.Background())).To(Equal(request.WithContext(context.Background())))
			Expect(paramName).To(Equal("version"))
			Expect(len(logger.Logs())).To(Equal(1))
			Expect(logger.Logs()[0].Message).To(ContainSubstring("v1"))
		})
	})

	Context("when the version requested does not match any of the handlers", func() {
		BeforeEach(func() {
			fakeRataAdapter.ParamReturns("v100")
		})
		It("Rejects the request with a 406 status code", func() {
			MakeRequestWithLogger(checkVersionHandler.ServeHTTP, resp, request, logger)

			Expect(v0Count).To(Equal(0))
			Expect(v1Count).To(Equal(0))

			Expect(fakeErrorResponse.NotAcceptableCallCount()).To(Equal(1))
			l, _, err, desc := fakeErrorResponse.NotAcceptableArgsForCall(0)
			Expect(l).To(Equal(expectedLogger))
			Expect(err).To(BeNil())
			Expect(desc).To(Equal("api version 'v100' not supported"))
		})
	})
})
