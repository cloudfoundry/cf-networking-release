package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"proxy/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventuallyFailHandler", func() {
	var h *handlers.FlapHandler
	var requestCount int

	testResponseStatusCode := func(expectedStatusCode int) {
		requestCount++
		resp := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/flap", nil)
		Expect(err).NotTo(HaveOccurred())
		h.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(expectedStatusCode), fmt.Sprintf("failed on request %d", requestCount))
	}

	Context("when FlapInterval is not set", func() {
		It("always succeeds", func() {
			h = &handlers.FlapHandler{}
			for i := 1; i < 5; i++ {
				testResponseStatusCode(http.StatusOK)
			}
		})
	})

	Context("when FlapInterval is set to 1", func() {
		It("alternates between succeeding and failing", func() {
			h = &handlers.FlapHandler{FlapInterval: 1}
			for i := 1; i < 5; i++ {
				testResponseStatusCode(http.StatusOK)
				testResponseStatusCode(http.StatusInternalServerError)
			}
		})
	})

	Context("when FlapInterval is set to 2", func() {
		It("succeeds... and then fails after the 2nd request... and repeats", func() {
			h = &handlers.FlapHandler{FlapInterval: 2}
			for i := 1; i < 5; i++ {
				testResponseStatusCode(http.StatusOK)
				testResponseStatusCode(http.StatusOK)
				testResponseStatusCode(http.StatusInternalServerError)
				testResponseStatusCode(http.StatusInternalServerError)
			}
		})
	})

	Context("when FlapInterval is set to 3", func() {
		It("succeeds... and then fails after the 3nd request... and repeats", func() {
			h = &handlers.FlapHandler{FlapInterval: 3}
			for i := 1; i < 5; i++ {
				testResponseStatusCode(http.StatusOK)
				testResponseStatusCode(http.StatusOK)
				testResponseStatusCode(http.StatusOK)
				testResponseStatusCode(http.StatusInternalServerError)
				testResponseStatusCode(http.StatusInternalServerError)
				testResponseStatusCode(http.StatusInternalServerError)
			}
		})
	})
})
