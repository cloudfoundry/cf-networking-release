package middleware_test

import (
	"code.cloudfoundry.org/policy-server/middleware"

	"github.com/tedsuo/rata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("cors", func() {
	var (
		cors middleware.CORS
	)

	Describe("add option routes to existing rata routes struct", func() {
		Context("when provided rata routes", func() {
			var (
				rataRoutes rata.Routes
			)

			BeforeEach(func() {
				rataRoutes = rata.Routes{
					{Name: "uptime", Method: "GET", Path: "/"},
					{Name: "health", Method: "GET", Path: "/health"},
				}
			})

			It("adds a OPTIONS method for each route", func() {
				rataWithOptions := cors.AddOptionsRoutes("options", rataRoutes)
				Expect(rataWithOptions).To(ConsistOf(rata.Routes{
					{Name: "uptime", Method: "GET", Path: "/"},
					{Name: "options", Method: "OPTIONS", Path: "/"},
					{Name: "health", Method: "GET", Path: "/health"},
					{Name: "options", Method: "OPTIONS", Path: "/health"},
				}))
			})
		})

		Context("when provided a route with multiple methods", func() {
			var (
				rataRoutes rata.Routes
			)

			BeforeEach(func() {
				rataRoutes = rata.Routes{
					{Name: "networking", Method: "GET", Path: "/networking"},
					{Name: "networking", Method: "POST", Path: "/networking"},
				}
			})

			It("adds an OPTIONS method per path", func() {
				rataWithOptions := cors.AddOptionsRoutes("options", rataRoutes)
				Expect(rataWithOptions).To(ConsistOf(rata.Routes{
					{Name: "networking", Method: "GET", Path: "/networking"},
					{Name: "networking", Method: "POST", Path: "/networking"},
					{Name: "options", Method: "OPTIONS", Path: "/networking"},
				}))
			})
		})
	})
})
