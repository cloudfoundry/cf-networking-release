package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("XXSSProtectionHandler", func() {
	var (
		fakeHandler http.Handler

		xxssProtectionHandler handlers.XXSSProtectionHandler
		wrappedHandler        http.Handler
	)

	BeforeEach(func() {
		fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("some-handler"))
		})

		xxssProtectionHandler = handlers.XXSSProtectionHandler{}
		wrappedHandler = xxssProtectionHandler.Wrap(fakeHandler)
	})

	It("adds the x-xss-proection header to the response", func() {

		resp := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/", nil)

		wrappedHandler.ServeHTTP(resp, request)
		Expect(resp.Header().Get("X-XSS-Protection")).To(Equal("1; mode=block"))
	})
})
