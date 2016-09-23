package testsupport_test

import (
	"lib/testsupport"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running work in parallel", func() {
	var runner *testsupport.ParallelRunner

	BeforeEach(func() {
		runner = &testsupport.ParallelRunner{
			NumWorkers: 4,
		}
	})

	It("calls the workFunc for every item", func() {
		items := []int{1, 2, 3, 4}

		var callCount int32

		workFunc := func(item int) {
			atomic.AddInt32(&callCount, 1)
		}

		runner.RunOnSlice(items, workFunc)
		Expect(callCount).To(Equal(int32(4)))
	})

	It("runs some workFuncs in parallel", func() {
		items := []int{1, 2, 3, 4}

		callCount := new(int32)
		allowProgress := make(chan bool)

		workFunc := func(item int) {
			atomic.AddInt32(callCount, 1)

			<-allowProgress
		}

		go runner.RunOnSlice(items, workFunc)
		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(4)))

		for range items { // allow the workFuncs to finish
			allowProgress <- false
		}
	})

	PIt("runs no more than NumWorkers in parallel", func() {
		items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

		callCount := new(int32)
		allowProgress := make(chan bool)

		workFunc := func(item int) {
			atomic.AddInt32(callCount, 1)

			<-allowProgress
		}

		go runner.RunOnSlice(items, workFunc)
		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(runner.NumWorkers)))

		for range items { // unblock all work funcs
			allowProgress <- false
		}

		// check that all items were processed
		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(len(items))))
	})
})
