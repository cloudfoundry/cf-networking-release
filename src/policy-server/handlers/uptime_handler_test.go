package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UptimeHandler", func() {
	var (
		request *http.Request
		handler *handlers.UptimeHandler
		resp    *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/test", bytes.NewBuffer([]byte{}))
		Expect(err).NotTo(HaveOccurred())

		handler = &handlers.UptimeHandler{}
		resp = httptest.NewRecorder()
	})

	It("reports the uptime of the server", func() {
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(ContainSubstring("Network policy server, up for"))
	})
})
