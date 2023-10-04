package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"proxy/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EchoSourceIPHandler", func() {
	var (
		handler *handlers.DumpRequestHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
	)

	BeforeEach(func() {
		handler = &handlers.DumpRequestHandler{}
		resp = httptest.NewRecorder()
	})

	Describe("GET", func() {
		BeforeEach(func() {
			var err error
			req, err = http.NewRequest("GET", "/dumprequest", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a body with a dump of the request", func() {
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(ContainSubstring("GET /dumprequest"))
		})

		It("does not send debug headers by default", func() {
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Result().Header.Get("X-Proxy-Settable-Debug-Header")).To(BeEmpty())
			Expect(resp.Result().Header.Get("X-Proxy-Immutable-Debug-Header")).To(BeEmpty())
		})

		Context("when the returnHeaders query param is 'true'", func() {
			BeforeEach(func() {
				var err error
				req, err = http.NewRequest("GET", "/dumprequest?returnHeaders=true", nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add("X-Random-Header", "apple")
				req.Header.Add("X-Random-Header", "strawberry")
				req.Header.Add("X-Random-Header", "guava")
			})

			It("returns headers for inspection", func() {
				handler.ServeHTTP(resp, req)

				By("returning the debug headers", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(resp.Result().Header.Get("X-Proxy-Settable-Debug-Header")).To(Equal("default-settable-value-from-within-proxy-src-code"))
					Expect(resp.Result().Header.Get("X-Proxy-Immutable-Debug-Header")).To(Equal("default-immutable-value-from-within-proxy-src-code"))
				})

				By("and cloning other headers sent from the client", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					xRandomHeaderValues := resp.Result().Header.Values("X-Random-Header")
					Expect(len(xRandomHeaderValues)).To(Equal(3))
					Expect(xRandomHeaderValues).To(ContainElements("guava", "apple", "strawberry"))
				})
			})

			Context("and the client sends an 'X-Proxy-Settable-Debug-Header' in their request", func() {
				BeforeEach(func() {
					var err error
					req, err = http.NewRequest("GET", "/dumprequest?returnHeaders=true", nil)
					req.Header.Add("X-Proxy-Settable-Debug-Header", "rutabaga")
					Expect(err).NotTo(HaveOccurred())
				})

				It("respects the value from the header and returns it", func() {
					handler.ServeHTTP(resp, req)

					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(resp.Result().Header.Get("X-Proxy-Immutable-Debug-Header")).To(Equal("default-immutable-value-from-within-proxy-src-code"))

					xProxySettabbleDebugHeaderValues := resp.Result().Header.Values("X-Proxy-Settable-Debug-Header")
					Expect(len(xProxySettabbleDebugHeaderValues)).To(Equal(1))
					Expect(xProxySettabbleDebugHeaderValues).To(ContainElement("rutabaga"))
				})
			})
		})
	})
})
