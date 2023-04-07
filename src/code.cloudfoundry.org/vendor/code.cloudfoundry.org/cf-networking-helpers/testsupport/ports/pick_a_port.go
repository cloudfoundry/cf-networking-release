package ports

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
)

var (
	lastPortUsed int
	mutex        sync.Mutex
	once         sync.Once
)

// PickAPort returns a port that is likely free for use in a Ginkgo test
func PickAPort() int {
	mutex.Lock()
	defer mutex.Unlock()

	if lastPortUsed == 0 {
		once.Do(func() {
			const portRangeStart = 18000
			lastPortUsed = portRangeStart + GinkgoParallelProcess()*200
		})
	}

	lastPortUsed += 1
	return lastPortUsed
}
