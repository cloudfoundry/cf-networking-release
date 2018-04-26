package routes_test

import (
	"service-discovery-controller/routes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsRecorder", func() {
	var (
		metricsRecorder *routes.MetricsRecorder
	)

	BeforeEach(func() {
		metricsRecorder = &routes.MetricsRecorder{}
	})

	Context("when requests are recorded", func() {
		BeforeEach(func() {
			metricsRecorder.RecordRequest()
			metricsRecorder.RecordRequest()
		})

		It("has the metrics getter func return the request count", func() {
			count, err := metricsRecorder.Getter()
			Expect(err).NotTo(HaveOccurred())
			Expect(int(count)).To(Equal(2))
		})
	})

	Context("when the getter is called multiple times", func() {
		BeforeEach(func() {
			metricsRecorder.RecordRequest()
			metricsRecorder.RecordRequest()
		})

		It("only returns the count for the number of requests received since getter was last called", func() {
			count, err := metricsRecorder.Getter()
			Expect(err).NotTo(HaveOccurred())
			Expect(int(count)).To(Equal(2))

			count, err = metricsRecorder.Getter()
			Expect(err).NotTo(HaveOccurred())
			Expect(int(count)).To(Equal(0))
		})
	})

	Context("concurrency", func() {
		BeforeEach(func() {
			go func() {
				metricsRecorder.RecordRequest()
				metricsRecorder.RecordRequest()
			}()
		})

		It("should not race", func() {
			Eventually(func() float64 {
				count, err := metricsRecorder.Getter()
				Expect(err).ToNot(HaveOccurred())
				return count
			}, "2s").Should(Equal(float64(2)))
		})
	})

})
