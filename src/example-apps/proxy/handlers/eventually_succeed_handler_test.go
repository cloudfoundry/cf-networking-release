package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"proxy/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventuallySucceedHandler", func() {
	var h *handlers.EventuallySucceedHandler

	succeedAfter := func(count int) {
		for i := 0; i < count; i++ {
			resp := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/eventuallysucceed", nil)
			Expect(err).NotTo(HaveOccurred())
			h.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		}

		resp := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/eventuallysucceed", nil)
		Expect(err).NotTo(HaveOccurred())
		h.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK))
	}

	Context("when SucceedAfterCount is set to 5", func() {
		It("fails... and then succeeds after the 5th request", func() {
			h = &handlers.EventuallySucceedHandler{SucceedAfterCount: 5}
			succeedAfter(5)
		})
	})

	Context("when SucceedAfterCount is not set", func() {
		It("always succeeds", func() {
			h = &handlers.EventuallySucceedHandler{}
			succeedAfter(0)
		})
	})
})
