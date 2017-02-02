package cleaner_test

import (
	"errors"
	"os"
	"policy-server/cleaner"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Poller", func() {
	Describe("Run", func() {
		var (
			logger  *lagertest.TestLogger
			p       *cleaner.Poller
			signals chan os.Signal
			ready   chan struct{}

			cycleCount uint64
			retChan    chan error
		)

		BeforeEach(func() {
			signals = make(chan os.Signal)
			ready = make(chan struct{})

			cycleCount = 0
			retChan = make(chan error)

			logger = lagertest.NewTestLogger("test")

			p = &cleaner.Poller{
				Logger:       logger,
				PollInterval: 10 * time.Millisecond,

				SingleCycleFunc: func() error {
					atomic.AddUint64(&cycleCount, 1)
					return nil
				},
			}
		})

		It("calls the single cycle func", func() {
			go func() {
				retChan <- p.Run(signals, ready)
			}()

			Eventually(ready).Should(BeClosed())
			Eventually(func() uint64 {
				return atomic.LoadUint64(&cycleCount)
			}).Should(BeNumerically(">", 0))

			Consistently(retChan).ShouldNot(Receive())

			signals <- os.Interrupt
			Eventually(retChan).Should(Receive(nil))
		})

		Context("when the cycle func errors", func() {
			BeforeEach(func() {
				p.SingleCycleFunc = func() error { return errors.New("banana") }
			})

			It("logs the error and continues", func() {
				go func() {
					retChan <- p.Run(signals, ready)
				}()

				Eventually(ready).Should(BeClosed())

				Eventually(logger).Should(gbytes.Say("poll-cycle.*banana"))

				Consistently(retChan).ShouldNot(Receive())

				signals <- os.Interrupt
				Eventually(retChan).Should(Receive(nil))
			})
		})
	})
})
