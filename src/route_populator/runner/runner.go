package runner

import (
	"errors"
	"fmt"
	"route_populator/publisher"
	"sync"
	"time"
)

type Runner struct {
	stopped bool

	cc  publisher.ConnectionCreator
	job publisher.Job

	numGoRoutines     int
	heartbeatInterval time.Duration
	publishDelay      time.Duration

	wg *sync.WaitGroup

	errsChan chan error
	quitChan chan struct{}
}

func NewRunner(c publisher.ConnectionCreator, j publisher.Job, numGoRoutines int, heartbeatInterval time.Duration, publishDelay time.Duration) *Runner {
	return &Runner{
		cc:                c,
		job:               j,
		numGoRoutines:     numGoRoutines,
		wg:                &sync.WaitGroup{},
		errsChan:          make(chan error, numGoRoutines),
		quitChan:          make(chan struct{}, 1),
		heartbeatInterval: heartbeatInterval,
		publishDelay:      publishDelay,
	}
}

func (r *Runner) Start() error {
	if r.stopped {
		return errors.New("Cannot restart a runner.")
	}

	numRoutes := r.job.EndRange - r.job.StartRange
	rangeSize := numRoutes / r.numGoRoutines
	ranges := PartitionRange(r.job.StartRange, r.job.EndRange, rangeSize)

	for i := 0; i < r.numGoRoutines; i += 1 {
		r.wg.Add(1)
		go func(id int) {
			defer r.wg.Done()

			job := r.job
			job.StartRange = ranges[id]
			job.EndRange = ranges[id+1]
			p := publisher.NewPublisher(job, r.publishDelay)
			err := p.Initialize(r.cc)
			if err != nil {
				r.errsChan <- fmt.Errorf("initializing connection: %s", err)
				r.Stop()
				return
			}
			err = p.PublishRouteRegistrations()
			if err != nil {
				r.errsChan <- fmt.Errorf("publishing: %s", err)
				r.Stop()
				return
			}
			for {
				select {
				case <-time.After(r.heartbeatInterval):
					err := p.PublishRouteRegistrations()
					if err != nil {
						r.errsChan <- fmt.Errorf("publishing: %s", err)
						r.Stop()
						return
					}
				case <-r.quitChan:
					// Exit upon closed quit channel
					return
				}
			}
		}(i)
	}

	return nil
}

func (r *Runner) Wait() error {
	r.wg.Wait()

	if len(r.errsChan) > 0 {
		err := <-r.errsChan
		return err
	}
	return nil
}

func (r *Runner) Stop() {
	if r.stopped == false {
		r.stopped = true
		close(r.quitChan)
	}
}

func min(a, b int) int {
	if b >= a {
		return a
	}
	return b
}

func PartitionRange(start, end, partitionSize int) []int {
	var result []int
	rangeSize := (end - start)
	if partitionSize == rangeSize {
		return []int{start, end}
	}
	for i := start; i <= end; i += partitionSize {
		result = append(result, i)
	}
	if (rangeSize % partitionSize) > 0 {
		result[len(result)-1] = end
	}
	return result
}
