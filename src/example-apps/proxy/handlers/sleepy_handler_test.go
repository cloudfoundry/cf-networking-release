package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"proxy/handlers"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SleepyHandler", func() {
	var h *handlers.SleepyHandler

	Context("when the client timeout is greater than the sleepy timeout", func() {
		It("succeeds", func() {
			h = &handlers.SleepyHandler{SleepyInterval: 5}

			ts := httptest.NewServer(h)

			client := &http.Client{
				Timeout: time.Second * 10,
			}
			resp, err := client.Get(ts.URL)
			Expect(err).NotTo(HaveOccurred())
			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("😴"))
		})
	})

	Context("when the client timeout is less than the sleepy timeout", func() {
		FIt("fails", func() {
			h = &handlers.SleepyHandler{SleepyInterval: 5}

			ts := httptest.NewServer(h)

			client := &http.Client{
				Timeout: time.Second * 4,
			}
			_, err := client.Get(ts.URL)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context deadline exceeded"))

		})
	})
})
