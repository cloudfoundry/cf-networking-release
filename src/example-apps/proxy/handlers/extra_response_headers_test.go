package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"proxy/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExtraResponseHeadersHandler", func() {
	var (
		h      *handlers.ExtraResponseHeadersHandler
		ts     *httptest.Server
		client *http.Client
	)

	BeforeEach(func() {
		h = &handlers.ExtraResponseHeadersHandler{}
		ts = httptest.NewServer(h)
		client = &http.Client{}
	})

	Context("when a number is passed", func() {
		It("adds extra headers", func() {

			By("first getting the default number of headers")
			resp, err := client.Get(ts.URL + "/extraresponseheaders/0")
			Expect(err).NotTo(HaveOccurred())
			defaultNumberOfHeaders := len(resp.Header)
			Expect(defaultNumberOfHeaders).To(BeNumerically(">", 1))
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("checking that the number of headers increased by the correct amount")
			extraHeadersToAdd := 10
			resp, err = client.Get(ts.URL + fmt.Sprintf("/extraresponseheaders/%d", extraHeadersToAdd))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resp.Header)).To(Equal(defaultNumberOfHeaders + extraHeadersToAdd))
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("checking that it works even with a trailing slash")
			resp, err = client.Get(ts.URL + fmt.Sprintf("/extraresponseheaders/%d/", extraHeadersToAdd))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resp.Header)).To(Equal(defaultNumberOfHeaders + extraHeadersToAdd))
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Context("when a number is not passed", func() {
		It("returns a status code 400 and logs an error", func() {
			resp, err := client.Get(ts.URL + "/extraresponseheaders/meow/")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})
