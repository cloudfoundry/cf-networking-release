package mbus_test

import (
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/service-discovery-controller/mbus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsRecorder", func() {
	var (
		recorder *mbus.MetricsRecorder
	)

	BeforeEach(func() {
		currentSystemTime := time.Unix(0, secondToNanosecond(810))
		fakeClock := fakeclock.NewFakeClock(currentSystemTime)
		recorder = &mbus.MetricsRecorder{
			Clock: fakeClock,
		}
	})

	It("should return the highest value since the last time it was asked", func() {
		recorder.RecordMessageTransitTime(secondToNanosecond(807))
		recorder.RecordMessageTransitTime(secondToNanosecond(809))
		recorder.RecordMessageTransitTime(secondToNanosecond(808))

		maxTime, err := recorder.GetMaxSinceLastInterval()
		Expect(err).NotTo(HaveOccurred())
		Expect(maxTime).To(Equal(float64(3000)))
	})

	It("should not record zero unix times", func() {
		recorder.RecordMessageTransitTime(0)

		time, err := recorder.GetMaxSinceLastInterval()
		Expect(err).NotTo(HaveOccurred())
		Expect(time).To(Equal(float64(0)))
	})

	It("should return the correct number of register messages received", func() {
		recorder.RecordRegisterMessageReceived()
		Expect(recorder.GetRegisterMessagesReceived()).To(Equal(float64(1)))

		recorder.RecordRegisterMessageReceived()
		recorder.RecordRegisterMessageReceived()
		recorder.RecordRegisterMessageReceived()
		Expect(recorder.GetRegisterMessagesReceived()).To(Equal(float64(4)))
	})
})

func secondToNanosecond(sec int) int64 {
	duration := time.Duration(sec) * time.Second
	return duration.Nanoseconds()
}
