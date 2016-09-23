package testsupport_test

import (
	"lib/testsupport"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running work in parallel", func() {
	var runner *testsupport.ParallelRunner

	BeforeEach(func() {
		runner = &testsupport.ParallelRunner{
			NumWorkers: 4,
			Timeout:    500 * time.Millisecond,
		}
	})

	It("calls the workFunc for every item", func() {
		items := []interface{}{1, 2, 3, 4}

		callCount := new(int32)

		workFunc := func(item interface{}) {
			atomic.AddInt32(callCount, 1)
		}

		go runner.RunOnSlice(items, workFunc)
		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(4)))
	})

	It("runs some workFuncs in parallel", func() {
		items := []interface{}{1, 2, 3, 4}

		callCount := new(int32)
		allowProgress := make(chan bool)

		workFunc := func(item interface{}) {
			atomic.AddInt32(callCount, 1)

			<-allowProgress
		}

		go runner.RunOnSlice(items, workFunc)
		Eventually(func() int32 { return atomic.LoadInt32(callCount) }).Should(
			Equal(int32(4)))

		for range items {
			allowProgress <- false
		}
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

	It("will timeout eventually if a worker hangs", func() {
		items := []interface{}{1, 2, 3, 4}

		workFunc := func(item interface{}) {
			// bad slow workers
			time.Sleep(60 * time.Second)
		}

		err := runner.RunOnSlice(items, workFunc)
		Expect(err).To(MatchError("timeout waiting for workers"))
	})

})
