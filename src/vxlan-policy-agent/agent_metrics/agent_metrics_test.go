package agent_metrics_test

import (
	"vxlan-policy-agent/agent_metrics"
	"vxlan-policy-agent/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent Metrics", func() {

	var fakeTimer *fakes.Timer
	BeforeEach(func() {
		fakeTimer = &fakes.Timer{}
		fakeTimer.ElapsedTimeReturns(100.0, nil)
	})

	Describe("NewElapsedTimeMetricSource", func() {
		It("returns the elapsed time", func() {
			source := agent_metrics.NewElapsedTimeMetricSource(fakeTimer, "elapsedTime")
			Expect(source.Name).To(Equal("elapsedTime"))
			Expect(source.Unit).To(Equal("ms"))

			value, err := source.Getter()
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeTimer.ElapsedTimeCallCount()).To(Equal(1))
			Expect(value).To(Equal(100.0))
		})
	})

	Describe("Timer", func() {
		var timer agent_metrics.Timer
		BeforeEach(func() {
			timer = agent_metrics.Timer{}
		})
		It("takes start and end times in nanoseconds and returns the elapsed time in milliseconds", func() {
			value, err := timer.ElapsedTime(20*1e9, 30*1e9)
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(10 * 1e3))
		})
	})
})
