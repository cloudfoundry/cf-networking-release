package planner_test

import (
	"time"
	"vxlan-policy-agent/planner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoggingState", func() {
	It("is disabled by default", func() {
		loggingState := &planner.LoggingState{}
		Expect(loggingState.IsEnabled()).To(BeFalse())
	})

	It("can be enabled and disabled", func() {
		loggingState := &planner.LoggingState{}
		loggingState.Enable()
		Expect(loggingState.IsEnabled()).To(BeTrue())
		loggingState.Disable()
		Expect(loggingState.IsEnabled()).To(BeFalse())
	})

	It("can be safely accessed by concurrent goroutines", func() {
		// run this test with --race
		loggingState := &planner.LoggingState{}

		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					loggingState.Enable()
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()

		for i := 0; i < 100; i++ {
			loggingState.IsEnabled()
			time.Sleep(10 * time.Millisecond)
		}
		close(done)

	})
})
