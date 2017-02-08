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
		request            *http.Request
		resp               *httptest.ResponseRecorder
		innerHandler       *fakes.HTTPHandler
		outerHandler       http.Handler
		metricWrapper      *handlers.MetricWrapper
		fakeMetricsEmitter *fakes.MetricsEmitter
	)
	Describe("Wrap", func() {
		BeforeEach(func() {
			fakeMetricsEmitter = &fakes.MetricsEmitter{}
			metricWrapper = &handlers.MetricWrapper{
				Name:           "name",
				MetricsEmitter: fakeMetricsEmitter,
			}
			var err error

			request, err = http.NewRequest("GET", "asdf", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())

			innerHandler = &fakes.HTTPHandler{}
			outerHandler = metricWrapper.Wrap(innerHandler)
		})

		It("emits a metric", func() {
			outerHandler.ServeHTTP(resp, request)
			Expect(fakeMetricsEmitter.EmitDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsEmitter.EmitDurationArgsForCall(0)
			Expect(name).To(Equal("name"))
		})

		It("serves the request with the provided handler", func() {
			outerHandler.ServeHTTP(resp, request)
			Expect(innerHandler.ServeHTTPCallCount()).To(Equal(1))
		})
	})
})
