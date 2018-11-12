package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"

	"github.com/tedsuo/rata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
				{Name: "networking", Method: "GET", Path: "/networking/:version/external/policies"},
				{Name: "networking", Method: "POST", Path: "/networking/:version/external/policies"},
			}

			fakeHandlerCallCount = 0
			fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				fakeHandlerCallCount++
				w.Write([]byte("some-handler"))
			})

			corsOptionsWrapper = handlers.CORSOptionsWrapper{
				RataRoutes:         rataRoutes,
				AllowedCORSDomains: allowedCORSDomains,
			}

			corsOptionsHandler = corsOptionsWrapper.Wrap(fakeHandler)
		})

		DescribeTable("adds Access-Control-Allow-Origin to header for the request origin", func(origin string, expectedAllowOrigins []string) {
			resp := httptest.NewRecorder()
			request, _ := http.NewRequest("GET", "/", nil)
			request.Header.Add("origin", origin)

			corsOptionsHandler.ServeHTTP(resp, request)
			Expect(resp.Header()["Access-Control-Allow-Origin"]).To(Equal(expectedAllowOrigins))
			Expect(resp.Header()).NotTo(HaveKey("Access-Control-Allow-Methods"))
		},
			Entry("when the origin is https://foo.bar", "https://foo.bar", []string{"https://foo.bar"}),
			Entry("when the origin is https://bar.foo", "https://bar.foo", []string{"https://bar.foo"}),
			Entry("when the origin is not in the allowed list", "https://bing.com", nil),
		)

		Context("when the allowed domains includes a '*'", func() {
			BeforeEach(func() {
				corsOptionsWrapper = handlers.CORSOptionsWrapper{
					RataRoutes:         rataRoutes,
					AllowedCORSDomains: []string{"*"},
				}

				corsOptionsHandler = corsOptionsWrapper.Wrap(fakeHandler)
			})

			It("allows any domain", func() {
				resp := httptest.NewRecorder()
				request, _ := http.NewRequest("GET", "/", nil)
				request.Header.Add("origin", "https://bing.com")

				corsOptionsHandler.ServeHTTP(resp, request)
				Expect(resp.Header()["Access-Control-Allow-Origin"]).To(Equal([]string{"*"}))
			})
		})

		It("calls the wrapped handler", func() {
			resp := httptest.NewRecorder()
			request, _ := http.NewRequest("GET", "/networking/v1/external/policies", nil)

			corsOptionsHandler.ServeHTTP(resp, request)
			Expect(fakeHandlerCallCount).To(Equal(1))
		})

		It("returns an error when a malformed path is provided", func() {
			corsOptionsWrapper = handlers.CORSOptionsWrapper{
				RataRoutes: rata.Routes{
					{Name: "badroute", Method: "GET", Path: "+++"},
				},
				AllowedCORSDomains: allowedCORSDomains,
			}
			corsOptionsHandler = corsOptionsWrapper.Wrap(fakeHandler)

			resp := httptest.NewRecorder()
			request, _ := http.NewRequest("OPTIONS", "/", nil)

			corsOptionsHandler.ServeHTTP(resp, request)
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})

		Context("when request method is OPTIONS", func() {
			It("adds security related headers", func() {
				resp := httptest.NewRecorder()
				request, _ := http.NewRequest("OPTIONS", "/networking/v1/external/policies", nil)
				request.Header.Add("origin", "https://foo.bar")

				corsOptionsHandler.ServeHTTP(resp, request)

				Expect(resp.Header()["Access-Control-Allow-Methods"]).To(ContainElement("GET,POST"))
				Expect(resp.Header()["Access-Control-Allow-Headers"]).To(ContainElement("authorization"))
				Expect(resp.Header()["Access-Control-Allow-Origin"]).To(Equal([]string{"https://foo.bar"}))

				Expect(resp.Header()["X-Frame-Options"]).To(Equal([]string{"deny"}))
				Expect(resp.Header()["Content-Security-Policy"]).To(Equal([]string{"frame-ancestors 'none'"}))
			})
		})
	})
})
