package poller_test

import (
	"errors"
	"lib/rules"
	"os"
	"time"
	"vxlan-policy-agent/fakes"
	"vxlan-policy-agent/poller"

	libfakes "lib/fakes"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Poller", func() {
	Describe("Run", func() {
		var (
			logger             *lagertest.TestLogger
			p                  *poller.Poller
			c                  rules.Chain
			fakePlanner        *fakes.Planner
			fakeEnforcer       *libfakes.RuleEnforcer
			timeMetricsEmitter *fakes.TimeMetricsEmitter
			r                  []rules.Rule
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")

			c = rules.Chain{
				Table:       "some-table",
				ParentChain: "INPUT",
				Prefix:      "some-prefix",
			}

			fakePlanner = &fakes.Planner{}
			fakeEnforcer = &libfakes.RuleEnforcer{}
			timeMetricsEmitter = &fakes.TimeMetricsEmitter{}

			p = &poller.Poller{
				Logger:            logger,
				PollInterval:      1 * time.Millisecond,
				Planner:           fakePlanner,
				Chain:             c,
				Enforcer:          fakeEnforcer,
				CollectionEmitter: timeMetricsEmitter,
			}

			r = []rules.Rule{}
			fakePlanner.GetRulesReturns(r, nil)
		})

		It("enforces rules on configured interval", func() {
			signals := make(chan os.Signal)
			ready := make(chan struct{})
			go p.Run(signals, ready)
			Eventually(ready).Should(BeClosed())
			Eventually(fakePlanner.GetRulesCallCount()).Should(BeNumerically(">", 0))
			Eventually(fakeEnforcer.EnforceOnChainCallCount()).Should(BeNumerically(">", 0))
			signals <- os.Interrupt

			ch, rs := fakeEnforcer.EnforceOnChainArgsForCall(0)
			Expect(ch).To(Equal(c))
			Expect(rs).To(Equal(r))
		})
		It("emits time metrics", func() {
			signals := make(chan os.Signal)
			ready := make(chan struct{})
			go p.Run(signals, ready)
			Eventually(ready).Should(BeClosed())
			Expect(timeMetricsEmitter.EmitAllCallCount()).To(BeNumerically(">", 0))
			signals <- os.Interrupt
		})

		Context("when planner errors", func() {
			BeforeEach(func() {
				fakePlanner.GetRulesReturns(r, errors.New("eggplant"))
			})

			It("logs the error and continues", func() {
				signals := make(chan os.Signal)
				ready := make(chan struct{})
				go p.Run(signals, ready)

				Eventually(logger).Should(gbytes.Say("get-rules.*eggplant"))
				Consistently(fakeEnforcer.EnforceOnChainCallCount()).Should(Equal(0))
				signals <- os.Interrupt
			})
		})

		Context("when enforcer errors", func() {
			BeforeEach(func() {
				fakeEnforcer.EnforceOnChainReturns(errors.New("eggplant"))
			})

			It("logs the error and continues", func() {
				signals := make(chan os.Signal)
				ready := make(chan struct{})
				go p.Run(signals, ready)

				Eventually(logger).Should(gbytes.Say("enforce.*eggplant"))
				signals <- os.Interrupt
			})
		})
	})
})
