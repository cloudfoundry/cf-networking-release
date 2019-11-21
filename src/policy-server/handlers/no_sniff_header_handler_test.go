package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("No-sniff header handler", func() {
	var (
		fakeHandler http.Handler

		noSniffHeaderHandler handlers.NoSniffHeaderHandler
		wrappedHandler       http.Handler
	)

	BeforeEach(func() {
		fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("some-handler"))
		})

		noSniffHeaderHandler = handlers.NoSniffHeaderHandler{}
		wrappedHandler = noSniffHeaderHandler.Wrap(fakeHandler)
	})

	It("adds the x-content-type-options header to the response", func() {
		resp := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/", nil)

		wrappedHandler.ServeHTTP(resp, request)
		Expect(resp.Header().Get("x-content-type-options")).To(Equal("nosniff"))
	})
})
