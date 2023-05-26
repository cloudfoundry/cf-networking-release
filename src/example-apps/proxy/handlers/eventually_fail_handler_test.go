package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"proxy/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventuallyFailHandler", func() {

	It("succeeds... and then fails on the 6th request", func() {
		failAfter(5)
	})

	Context("when EVENTUALLY_FAIL_AFTER_COUNT is set", func() {
		It("succeeds... and then fails on the configured request number", func() {
			os.Setenv("EVENTUALLY_FAIL_AFTER_COUNT", "2")
			failAfter(2)
		})

		Context("when the EVENTUALLY_FAIL_AFTER_COUNT is not parseable as an int", func() {
			It("uses the default instead of the env var", func() {
				os.Setenv("EVENTUALLY_FAIL_AFTER_COUNT", "M30W!")
				failAfter(5)
			})
		})
	})
})

func failAfter(count int) {
	handler := &handlers.EventuallyFailHandler{}

	for i := 0; i < count; i++ {
		resp := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/eventuallyfail", nil)
		Expect(err).NotTo(HaveOccurred())
		handler.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK))
	}

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/eventuallyfail", nil)
	Expect(err).NotTo(HaveOccurred())
	handler.ServeHTTP(resp, req)
	Expect(resp.Code).To(Equal(http.StatusInternalServerError))
}
