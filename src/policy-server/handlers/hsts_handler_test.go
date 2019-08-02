package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HSTSHandler", func() {
	var (
		fakeHandler http.Handler

		hstsHandler    handlers.HSTSHandler
		wrappedHandler http.Handler
	)

	BeforeEach(func() {
		fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("some-handler"))
		})

		hstsHandler = handlers.HSTSHandler{}
		wrappedHandler = hstsHandler.Wrap(fakeHandler)
	})

	It("adds the Strict-Transport-Security header to the response", func() {

		resp := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/", nil)

		wrappedHandler.ServeHTTP(resp, request)
		Expect(resp.Header().Get("Strict-Transport-Security")).To(Equal("max-age=31536000"))
	})
})
