package testsupport

import (
	"fmt"
	"sync"
	"time"
)

type ParallelRunner struct {
	NumWorkers int
	Timeout    time.Duration
}

func (p *ParallelRunner) RunOnSlice(items []interface{}, workFunc func(item interface{})) error {
	queue := make(chan interface{})

	go func() {
		for _, item := range items {
			queue <- item
		}
		close(queue)
	}()

	return p.RunOnChannel(queue, workFunc)
}

func (p *ParallelRunner) RunOnChannel(queue chan interface{}, workFunc func(item interface{})) error {
	var wg sync.WaitGroup

	for i := 0; i < p.NumWorkers; i++ {
		wg.Add(1)
		go func() {
			for item := range queue {
				workFunc(item)
			}
			wg.Done()
		}()
	}

	done := make(chan interface{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(p.Timeout):
		return fmt.Errorf("timeout waiting for workers")
	}
}
