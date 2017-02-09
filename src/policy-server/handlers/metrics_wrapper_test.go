package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsWrapper", func() {
	var (
		request           *http.Request
		resp              *httptest.ResponseRecorder
		innerHandler      *fakes.HTTPHandler
		outerHandler      http.Handler
		metricWrapper     *handlers.MetricWrapper
		fakeMetricsSender *fakes.MetricsSender
	)
	Describe("Wrap", func() {
		BeforeEach(func() {
			fakeMetricsSender = &fakes.MetricsSender{}
			metricWrapper = &handlers.MetricWrapper{
				Name:          "name",
				MetricsSender: fakeMetricsSender,
			}

			var err error
			request, err = http.NewRequest("GET", "asdf", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())

			innerHandler = &fakes.HTTPHandler{}
			outerHandler = metricWrapper.Wrap(innerHandler)
		})

		It("emits a request duration metric", func() {
			outerHandler.ServeHTTP(resp, request)
			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("name"))
		})

		It("serves the request with the provided handler", func() {
			outerHandler.ServeHTTP(resp, request)
			Expect(innerHandler.ServeHTTPCallCount()).To(Equal(1))
		})
	})
})
