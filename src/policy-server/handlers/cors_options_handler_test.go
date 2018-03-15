package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"

	"github.com/tedsuo/rata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CORS Option Handler", func() {
	var (
		rataRoutes           rata.Routes
		allowedCORSDomains   []string
		fakeHandler          http.Handler
		fakeHandlerCallCount int

		corsOptionsWrapper handlers.CORSOptionsWrapper
		corsOptionsHandler http.Handler
	)

	Context("when allowed cors domain and rata routes are provided", func() {
		BeforeEach(func() {
			allowedCORSDomains = []string{
				"https://foo.bar",
				"https://bar.foo",
			}
			rataRoutes = rata.Routes{
				{Name: "uptime", Method: "GET", Path: "/"},
				{Name: "networking", Method: "GET", Path: "/networking"},
				{Name: "networking", Method: "POST", Path: "/networking"},
			}

			fakeHandlerCallCount = 0
			fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				fakeHandlerCallCount++
				w.Write([]byte("some-handler"))
			})

			corsOptionsWrapper = handlers.CORSOptionsWrapper{
				RataRoutes:       rataRoutes,
				AllowCORSDomains: allowedCORSDomains,
			}

			corsOptionsHandler = corsOptionsWrapper.Wrap(fakeHandler)
		})

		It("adds Access-Control-Allow-Origin to header", func() {
			resp := httptest.NewRecorder()
			request, _ := http.NewRequest("GET", "/", nil)

			corsOptionsHandler.ServeHTTP(resp, request)

			Expect(resp.Header()["Access-Control-Allow-Origin"]).To(ContainElement("https://foo.bar,https://bar.foo"))
			Expect(resp.Header()).NotTo(HaveKey("Access-Control-Allow-Methods"))
		})

		It("calls the wrapped handler", func() {
			resp := httptest.NewRecorder()
			request, _ := http.NewRequest("GET", "/networking", nil)

			corsOptionsHandler.ServeHTTP(resp, request)
			Expect(fakeHandlerCallCount).To(Equal(1))
		})

		Context("when request method is OPTIONS", func() {
			It("adds Access-Control-Allow-Methods to header", func() {
				resp := httptest.NewRecorder()
				request, _ := http.NewRequest("OPTIONS", "/networking", nil)

				corsOptionsHandler.ServeHTTP(resp, request)

				Expect(resp.Header()["Access-Control-Allow-Methods"]).To(ContainElement("GET,POST"))
			})
		})
	})
})
