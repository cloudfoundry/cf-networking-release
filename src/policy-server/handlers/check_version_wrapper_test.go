package handlers_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tedsuo/rata"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
)

var _ = Describe("CheckVersionWrapper", func() {
	var (
		checkVersionHandler http.HandlerFunc
		checkVersionWrapper *handlers.CheckVersionWrapper
		fakeHandlerv0       http.HandlerFunc
		fakeHandlerv1       http.HandlerFunc
		handlerMap          map[string]http.HandlerFunc
		server              *httptest.Server
		logger              *lagertest.TestLogger
		fakeErrorResponse   *fakes.ErrorResponse
		client              *http.Client

		v0Count int
		v1Count int
	)

	BeforeEach(func() {
		fakeHandlerv0 = func(w http.ResponseWriter, req *http.Request) {
			v0Count++
			logger.Info("v0")
			w.Write([]byte("v0"))
		}

		fakeHandlerv1 = func(w http.ResponseWriter, req *http.Request) {
			v1Count++
			logger.Info("v1")
			w.Write([]byte("v1"))
		}

		handlerMap = map[string]http.HandlerFunc{
			"v0": fakeHandlerv0,
			"v1": fakeHandlerv1,
		}

		fakeErrorResponse = &fakes.ErrorResponse{}

		logger = lagertest.NewTestLogger("test")

		checkVersionWrapper = &handlers.CheckVersionWrapper{ErrorResponse: fakeErrorResponse}
		checkVersionHandler = checkVersionWrapper.CheckVersion(handlerMap)
		routes := rata.Routes{
			{Name: "some_resource", Method: "GET", Path: "/networking/:version/some/resource"},
		}
		handlers := rata.Handlers{
			"some_resource": middleware.LogWrap(logger, checkVersionHandler),
		}
		router, err := rata.NewRouter(routes, handlers)
		Expect(err).NotTo(HaveOccurred())

		server = httptest.NewServer(router)
		client = http.DefaultClient
	})

	AfterEach(func() {
		v0Count = 0
		v1Count = 0
	})

	It("should delegate to handler of the requested version", func() {
		resp, err := client.Get(fmt.Sprintf("%s/networking/v1/some/resource", server.URL))
		Expect(err).NotTo(HaveOccurred())

		bytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(v0Count).To(Equal(0))
		Expect(v1Count).To(Equal(1))
		Expect(string(bytes)).To(Equal("v1"))
		Expect(len(logger.Logs())).To(BeNumerically(">", 1))
		Expect(logger.Logs()[1].Message).To(ContainSubstring("v1"))
	})

	Context("when the version requested does not match any of the handlers", func() {
		It("Rejects the request with a 406 status code", func() {
			resp, err := client.Get(fmt.Sprintf("%s/networking/v100/some/resource", server.URL))
			Expect(err).NotTo(HaveOccurred())

			bytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(v0Count).To(Equal(0))
			Expect(v1Count).To(Equal(0))
			Expect(bytes).To(BeEmpty())

			Expect(fakeErrorResponse.NotAcceptableCallCount()).To(Equal(1))
			_, err, message, desc := fakeErrorResponse.NotAcceptableArgsForCall(0)
			Expect(err).To(BeNil())
			Expect(message).To(Equal("check api version"))
			Expect(desc).To(Equal("api version 'v100' not supported"))
		})
	})
})
