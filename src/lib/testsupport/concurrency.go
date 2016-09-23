package testsupport

type ParallelRunner struct {
	NumWorkers int
}

func (p *ParallelRunner) RunOnSlice(items []int, workFunc func(item int)) {
	completedItems := make(chan int, len(items))

	for _, item := range items {
		go func(item int) {
			// TODO: defer GinkgoRecover()
			workFunc(item)
			completedItems <- item
		}(item)
	}

	for range items { // wait until all items are complete
		<-completedItems
	}
}
