package testsupport

import (
	"fmt"
	"time"

	"github.com/tedsuo/ifrit"
)

func WaitOrReady(startTimeout time.Duration, monitor ifrit.Process) error {
	select {
	case err := <-monitor.Wait():
		return err
	case <-monitor.Ready():
		return nil
	case <-time.After(startTimeout):
		return fmt.Errorf("timeout: ifrit failed to close ready channel")
	}
}
