package testsupport

import (
	"sync"

	. "github.com/onsi/ginkgo"
)

type ParallelRunner struct {
	NumWorkers int
}

func (p *ParallelRunner) RunOnSlice(items []interface{}, workFunc func(item interface{})) {
	queue := make(chan interface{})

	go func() {
		for _, item := range items {
			queue <- item
		}
		close(queue)
	}()

	p.RunOnChannel(queue, workFunc)
}

func (p *ParallelRunner) RunOnChannel(queue chan interface{}, workFunc func(item interface{})) {
	var wg sync.WaitGroup

	for i := 0; i < p.NumWorkers; i++ {
		wg.Add(1)
		go func() {
			defer GinkgoRecover() //not tested
			defer wg.Done()
			for item := range queue {
				workFunc(item)
			}
		}()
	}

	wg.Wait()
}

func (p *ParallelRunner) RunOnSliceStrings(someStrings []string, workFunc func(aString string)) {
	items := []interface{}{}
	for _, aString := range someStrings {
		items = append(items, aString)
	}

	p.RunOnSlice(items, func(item interface{}) { workFunc(item.(string)) })
}
