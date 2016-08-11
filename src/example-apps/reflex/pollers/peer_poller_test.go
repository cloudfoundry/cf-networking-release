package pollers_test

import (
	"errors"
	"example-apps/reflex/fakes"
	"example-apps/reflex/pollers"
	"os"
	"time"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PeerPoller", func() {
	var (
		logger    *lagertest.TestLogger
		poller    *pollers.PeerPoller
		converger *fakes.Converger
		timer     *fakes.Timer
		timeCh    chan time.Time
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		converger = &fakes.Converger{}
		timer = &fakes.Timer{}

		poller = &pollers.PeerPoller{
			Logger:    logger,
			Converger: converger,
			Timer:     timer,
		}

		timer.AfterStub = func() <-chan time.Time {
			timeCh = make(chan time.Time)
			return timeCh
		}
	})

	It("only receives from the timer channel once", func(done Done) {
		signals := make(chan os.Signal)
		ready := make(chan struct{})
		go poller.Run(signals, ready)
		Eventually(ready).Should(BeClosed())
		Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(0))
		close(timeCh)
		Eventually(converger.ConvergeCallCount).Should(Equal(1))
		Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(1))
		signals <- os.Interrupt
		Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(1))
		close(done)
	}, 1.0 /* seconds to wait on the whole spec before failing */)

	It("closes the ready channel and then continuously calls the converger on the poll interval", func(done Done) {
		signals := make(chan os.Signal)
		ready := make(chan struct{})
		go poller.Run(signals, ready)
		Eventually(ready).Should(BeClosed())
		Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(0))
		timeCh <- time.Now()
		Eventually(converger.ConvergeCallCount).Should(Equal(1))
		Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(1))
		timeCh <- time.Now()
		Eventually(converger.ConvergeCallCount).Should(Equal(2))
		Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(2))
		signals <- os.Interrupt
		Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(2))
		close(done)
	}, 1.0 /* seconds to wait on the whole spec before failing */)

	Context("when a signal arrives before a timer tick", func() {
		It("returns early and does not call Converge anymore", func(done Done) {
			signals := make(chan os.Signal, 1)
			ready := make(chan struct{})
			finished := make(chan error)
			go func() {
				finished <- poller.Run(signals, ready)
			}()
			Eventually(ready).Should(BeClosed())
			Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(0))
			timeCh <- time.Now()
			Eventually(converger.ConvergeCallCount).Should(Equal(1))
			Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(1))

			signals <- os.Interrupt
			Eventually(finished).Should(Receive(nil))
			Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(1))
			close(done)
		}, 2.0 /* seconds to wait on the whole spec before failing */)
	})

	Context("when the converger returns an error", func() {
		BeforeEach(func() {
			converger.ConvergeReturns(errors.New("banana"))
		})
		It("logs it and continues polling", func(done Done) {
			signals := make(chan os.Signal)
			ready := make(chan struct{})
			go poller.Run(signals, ready)
			Consistently(converger.ConvergeCallCount, "100ms").Should(Equal(0))

			timeCh <- time.Now()
			Eventually(converger.ConvergeCallCount).Should(Equal(1))
			Eventually(logger).Should(gbytes.Say("error.*banana"))

			signals <- os.Interrupt
			close(done)
		}, 3.0 /* seconds to wait on the whole spec before failing */)
	})
})
