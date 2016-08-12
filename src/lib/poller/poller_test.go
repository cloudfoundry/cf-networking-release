package poller_test

import (
	"errors"
	"lib/fakes"
	"lib/poller"
	"lib/rules"
	"os"
	"time"

	"code.cloudfoundry.org/lager/lagertest"

	common_fakes "lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Poller", func() {
	Describe("Run", func() {
		var (
			logger       *lagertest.TestLogger
			p            *poller.Poller
			c            rules.Chain
			fakePlanner  *fakes.Planner
			fakeEnforcer *common_fakes.RuleEnforcer
			r            []rules.Rule
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")

			c = rules.Chain{
				Table:       "some-table",
				ParentChain: "INPUT",
				Prefix:      "some-prefix",
			}

			fakePlanner = &fakes.Planner{}
			fakeEnforcer = &common_fakes.RuleEnforcer{}

			p = &poller.Poller{
				Logger:       logger,
				PollInterval: 1 * time.Millisecond,
				Planner:      fakePlanner,
				Chain:        c,
				Enforcer:     fakeEnforcer,
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
