package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"vxlan-policy-agent/fakes"
	"vxlan-policy-agent/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IptablesLogging", func() {
	var (
		iptHandler   *handlers.IPTablesLogging
		recorder     *httptest.ResponseRecorder
		loggingState *fakes.LoggingState
	)

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
		loggingState = &fakes.LoggingState{}
		iptHandler = &handlers.IPTablesLogging{
			LoggingState: loggingState,
		}
	})

	Describe("getting the state", func() {
		Context("when logging is enabled", func() {
			BeforeEach(func() {
				loggingState.IsEnabledReturns(true)
			})
			It("returns true", func() {
				req, err := http.NewRequest("GET", "/", nil)
				Expect(err).NotTo(HaveOccurred())

				iptHandler.ServeHTTP(recorder, req)
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(loggingState.IsEnabledCallCount()).To(Equal(1))
				Expect(recorder.Body.String()).To(MatchJSON(`{"enabled": true}`))
			})
		})

		Context("when logging is disabled", func() {
			BeforeEach(func() {
				loggingState.IsEnabledReturns(false)
			})
			It("returns false", func() {
				req, err := http.NewRequest("GET", "/", nil)
				Expect(err).NotTo(HaveOccurred())

				iptHandler.ServeHTTP(recorder, req)
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(loggingState.IsEnabledCallCount()).To(Equal(1))
				Expect(recorder.Body.String()).To(MatchJSON(`{"enabled": false}`))
			})
		})
	})

	Describe("setting the logging state", func() {
		Context("when called with enabled: true", func() {
			It("sets the logging state", func() {
				req, err := http.NewRequest("PUT", "/", strings.NewReader(`{"enabled":true}`))
				Expect(err).NotTo(HaveOccurred())

				recorder := httptest.NewRecorder()
				iptHandler.ServeHTTP(recorder, req)
				Expect(recorder.Code).To(Equal(http.StatusOK))

				Expect(loggingState.EnableCallCount()).To(Equal(1))
			})
		})

		Context("when called with enabled: false", func() {
			It("sets the logging state", func() {
				req, err := http.NewRequest("PUT", "/", strings.NewReader(`{"enabled":false}`))
				Expect(err).NotTo(HaveOccurred())

				recorder := httptest.NewRecorder()
				iptHandler.ServeHTTP(recorder, req)
				Expect(recorder.Code).To(Equal(http.StatusOK))

				Expect(loggingState.DisableCallCount()).To(Equal(1))
			})
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
})
