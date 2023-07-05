package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"proxy/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventuallySucceedHandler", func() {

	It("fails... and then succeeds on the 6th request", func() {
		succeedAfter(5)
	})

	Context("when EVENTUALLY_SUCCEED_AFTER_COUNT is set", func() {
		It("fails... and then succeeds on the configured request number", func() {
			os.Setenv("EVENTUALLY_SUCCEED_AFTER_COUNT", "2")
			succeedAfter(2)
		})

		Context("when the EVENTUALLY_SUCCEED_AFTER_COUNT is not parseable as an int", func() {
			It("uses the default instead of the env var", func() {
				os.Setenv("EVENTUALLY_SUCCEED_AFTER_COUNT", "M30W!")
				succeedAfter(5)
			})
		})
	})
})

func succeedAfter(count int) {
	handler := &handlers.EventuallySucceedHandler{}

	for i := 0; i < count; i++ {
		resp := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/eventuallysucceed", nil)
		Expect(err).NotTo(HaveOccurred())
		handler.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusInternalServerError))
	}

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/eventuallysucceed", nil)
	Expect(err).NotTo(HaveOccurred())
	handler.ServeHTTP(resp, req)
	Expect(resp.Code).To(Equal(http.StatusOK))
}
