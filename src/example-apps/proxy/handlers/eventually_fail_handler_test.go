package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"proxy/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventuallyFailHandler", func() {
	var h *handlers.EventuallyFailHandler

	failAfter := func(count int) {
		for i := 0; i < count; i++ {
			resp := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/eventuallyfail", nil)
			Expect(err).NotTo(HaveOccurred())
			h.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))
		}

		resp := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/eventuallyfail", nil)
		Expect(err).NotTo(HaveOccurred())
		h.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
	}

	Context("when FailAfterCount is set to 5", func() {
		It("succeeds... and then fails after the 5th request", func() {
			h = &handlers.EventuallyFailHandler{FailAfterCount: 5}
			failAfter(5)
		})
	})

	Context("when FailAfterCount is not set", func() {
		It("always fails", func() {
			h = &handlers.EventuallyFailHandler{}
			failAfter(0)
		})
	})
})
