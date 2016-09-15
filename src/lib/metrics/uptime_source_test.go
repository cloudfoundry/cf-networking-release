package metrics_test

import (
	"lib/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UptimeSource", func() {
	It("reports the uptime since the source was created", func() {
		uptimeSource := metrics.NewUptimeSource()

		Expect(uptimeSource.Name).To(Equal("uptime"))
		Expect(uptimeSource.Unit).To(Equal("seconds"))

		Eventually(uptimeSource.Getter, "2s").Should(
			BeNumerically(">", 1))
	})
})
