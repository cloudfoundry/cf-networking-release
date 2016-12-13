package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"vxlan-policy-agent/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IptablesLogging", func() {
	var (
		iptHandler *handlers.IPTablesLogging
		recorder   *httptest.ResponseRecorder
		logging    chan bool
	)

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
		logging = make(chan bool, 1)
		iptHandler = &handlers.IPTablesLogging{
			LoggingChan: logging,
		}
	})

	It("persists state", func() {
		By("getting the current state")
		req, err := http.NewRequest("GET", "/", nil)
		Expect(err).NotTo(HaveOccurred())

		iptHandler.ServeHTTP(recorder, req)
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(MatchJSON(`{"enabled": false}`))

		By("putting new state")
		req, err = http.NewRequest("PUT", "/", strings.NewReader(`{"enabled":true}`))
		Expect(err).NotTo(HaveOccurred())

		recorder = httptest.NewRecorder()
		iptHandler.ServeHTTP(recorder, req)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		By("getting the new state")
		req, err = http.NewRequest("GET", "/", nil)
		Expect(err).NotTo(HaveOccurred())

		recorder = httptest.NewRecorder()
		iptHandler.ServeHTTP(recorder, req)
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(MatchJSON(`{"enabled":true}`))
	})

	It("sends a signal on a channel", func() {
		req, err := http.NewRequest("PUT", "/", strings.NewReader(`{"enabled":true}`))
		Expect(err).NotTo(HaveOccurred())

		recorder := httptest.NewRecorder()
		iptHandler.ServeHTTP(recorder, req)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		Expect(logging).To(Receive(BeTrue()))
	})

	Context("when decoding the request body fails", func() {
		It("returns 400", func() {
			req, err := http.NewRequest("PUT", "/", strings.NewReader(`not json`))
			Expect(err).NotTo(HaveOccurred())

			recorder = httptest.NewRecorder()
			iptHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("when the PUT body doesn't set the 'enabled' key", func() {
		It("returns 400 and a useful error", func() {
			req, err := http.NewRequest("PUT", "/", strings.NewReader(`{}`))
			Expect(err).NotTo(HaveOccurred())

			recorder = httptest.NewRecorder()
			iptHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			Expect(recorder.Body.String()).To(MatchJSON(`{"error": "missing required key 'enabled'"}`))
		})
	})
})
