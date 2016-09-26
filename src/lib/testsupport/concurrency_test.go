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
	Describe("RunOnSliceStrings", func() {
		It("does the right thing", func() {
			items := []string{"foo", "bar"}

			callCount := new(int32)

			workFunc := func(item string) {
				atomic.AddInt32(callCount, 1)
			}

			runner.RunOnSliceStrings(items, workFunc)
			Expect(*callCount).To(Equal(int32(2)))
		})
	})

	It("calls the workFunc for every item", func() {
		items := []interface{}{1, 2, 3, 4}

		callCount := new(int32)

		workFunc := func(item interface{}) {
			atomic.AddInt32(callCount, 1)
		}

		runner.RunOnSlice(items, workFunc)
		Expect(*callCount).To(Equal(int32(4)))
	})

	Specify("once the function returns, all of the work is complete", func() {
		items := []interface{}{1, 2, 3, 4}

		callCount := new(int32)

		workFunc := func(item interface{}) {
			atomic.AddInt32(callCount, 1)
		}

		runner.RunOnSlice(items, workFunc)
		Expect(*callCount).To(Equal(int32(4)))
	})

	It("runs some workFuncs in parallel", func() {
		items := []interface{}{1, 2, 3, 4}

		callCount := new(int32)
		allowProgress := make(chan bool)

		workFunc := func(item interface{}) {
			atomic.AddInt32(callCount, 1)

			<-allowProgress
		}

		callComplete := make(chan bool)
		go func() {
			runner.RunOnSlice(items, workFunc)
			callComplete <- true
		}()

		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(4)))

		for range items {
			allowProgress <- false
		}

		Eventually(callComplete).Should(Receive())
	})

	It("runs no more than NumWorkers in parallel", func() {
		items := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9}

		callCount := new(int32)
		allowProgress := make(chan bool)

		workFunc := func(item interface{}) {
			atomic.AddInt32(callCount, 1)

			<-allowProgress
		}

		go runner.RunOnSlice(items, workFunc)
		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(runner.NumWorkers)))

		for range items {
			allowProgress <- false
		}

		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(len(items))))
	})
})
