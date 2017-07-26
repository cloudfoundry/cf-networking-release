package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
)

var _ = Describe("CheckVersionWrapper", func() {
	var (
		checkVersionHandler middleware.LoggableHandlerFunc
		checkVersionWrapper *handlers.CheckVersionWrapper
		request             *http.Request
		fakeHandlerv0       *fakeLoggableHandler
		fakeHandlerv1       *fakeLoggableHandler
		handlerMap          map[string]middleware.LoggableHandlerFunc
		resp                *httptest.ResponseRecorder
		logger              *lagertest.TestLogger
		fakeErrorResponse   *fakes.ErrorResponse
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/some/resource", bytes.NewBuffer([]byte(`{}`)))
		Expect(err).NotTo(HaveOccurred())
		request.Header["Accept"] = []string{"1.2.3+policy-server-json"}

		fakeHandlerv0 = &fakeLoggableHandler{}
		fakeHandlerv1 = &fakeLoggableHandler{}
		handlerMap = map[string]middleware.LoggableHandlerFunc{
			"0.0.0": fakeHandlerv0.LoggableHandler,
			"1.2.3": fakeHandlerv1.LoggableHandler,
		}

		fakeErrorResponse = &fakes.ErrorResponse{}

		logger = lagertest.NewTestLogger("test")

		resp = httptest.NewRecorder()
		checkVersionWrapper = &handlers.CheckVersionWrapper{ErrorResponse: fakeErrorResponse}
		checkVersionHandler = checkVersionWrapper.CheckVersion(handlerMap)

	})

	It("should delegate to handler of the requested version", func() {
		checkVersionHandler(logger, resp, request)

		Expect(fakeHandlerv0.invocationCount).To(Equal(0))
		Expect(fakeHandlerv1.invocationCount).To(Equal(1))
		Expect(fakeHandlerv1.actualLogger).To(Equal(logger))
		Expect(fakeHandlerv1.actualWriter).To(Equal(resp))
		Expect(fakeHandlerv1.actualRequest).To(Equal(request))
	})

	Context("when no accept header is provided", func() {
		BeforeEach(func() {
			delete(request.Header, "Accept")
		})

		It("should use the v0.0.0 handler", func() {
			checkVersionHandler(logger, resp, request)

			Expect(fakeHandlerv1.invocationCount).To(Equal(0))
			Expect(fakeHandlerv0.invocationCount).To(Equal(1))
			Expect(fakeHandlerv0.actualLogger).To(Equal(logger))
			Expect(fakeHandlerv0.actualWriter).To(Equal(resp))
			Expect(fakeHandlerv0.actualRequest).To(Equal(request))
		})
	})

	Context("when the version requested does not match any of the handlers", func() {
		BeforeEach(func() {
			request.Header["Accept"] = []string{"6.2.3+policy-server-json"}
		})
		It("Rejects the request with a 406 status code", func() {
			checkVersionHandler(logger, resp, request)

			Expect(fakeErrorResponse.NotAcceptableCallCount()).To(Equal(1))
			rw, err, message, desc := fakeErrorResponse.NotAcceptableArgsForCall(0)
			Expect(rw).To(Equal(resp))
			Expect(err).To(BeNil())
			Expect(message).To(Equal("check api version"))
			Expect(desc).To(Equal("api version '6.2.3+policy-server-json' not supported"))
		})
	})

	Context("when multiple accept values are provided", func() {
		BeforeEach(func() {
			request.Header["Accept"] = []string{"0.0.0", "2.0.0"}
		})

		It("should return a sensible error", func() {
			checkVersionHandler(logger, resp, request)

			Expect(fakeErrorResponse.BadRequestCallCount()).To(Equal(1))
			rw, err, message, desc := fakeErrorResponse.BadRequestArgsForCall(0)
			Expect(rw).To(Equal(resp))
			Expect(err).To(BeNil())
			Expect(message).To(Equal("check api version"))
			Expect(desc).To(Equal("multiple accept headers not allowed"))
		})
	})

	Context("when the version is not valid", func() {
		BeforeEach(func() {
			request.Header["Accept"] = []string{"banana"}
		})
		It("returns a 406 error", func() {
			checkVersionHandler(logger, resp, request)

			Expect(fakeErrorResponse.NotAcceptableCallCount()).To(Equal(1))
		})
	})

	Context("when the handler map has a bad key", func() {
		BeforeEach(func() {
			badHandlerMap := map[string]middleware.LoggableHandlerFunc{
				"banana": fakeHandlerv0.LoggableHandler,
				"1.2.3":  fakeHandlerv1.LoggableHandler,
			}
			checkVersionWrapper := handlers.CheckVersionWrapper{ErrorResponse: fakeErrorResponse}
			checkVersionHandler = checkVersionWrapper.CheckVersion(badHandlerMap)
		})
		It("ignores it", func() {
			checkVersionHandler(logger, resp, request)

			Expect(fakeHandlerv0.invocationCount).To(Equal(0))
			Expect(fakeHandlerv1.invocationCount).To(Equal(1))
		})
	})
})

type fakeLoggableHandler struct {
	invocationCount int
	actualLogger    lager.Logger
	actualWriter    http.ResponseWriter
	actualRequest   *http.Request
}

func (f *fakeLoggableHandler) LoggableHandler(logger lager.Logger, w http.ResponseWriter, r *http.Request) {
	f.invocationCount++
	f.actualLogger = logger
	f.actualWriter = w
	f.actualRequest = r
}
