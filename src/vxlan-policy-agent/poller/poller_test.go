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
			fakePlanner        *fakes.Planner
			fakeEnforcer       *libfakes.RuleEnforcer
			timeMetricsEmitter *fakes.TimeMetricsEmitter
			rulesWithChain     rules.RulesWithChain
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")

			fakePlanner = &fakes.Planner{}
			fakeEnforcer = &libfakes.RuleEnforcer{}
			timeMetricsEmitter = &fakes.TimeMetricsEmitter{}

			p = &poller.Poller{
				Logger:            logger,
				PollInterval:      1 * time.Millisecond,
				Planner:           fakePlanner,
				Enforcer:          fakeEnforcer,
				CollectionEmitter: timeMetricsEmitter,
			}

			rulesWithChain = rules.RulesWithChain{
				Rules: []rules.Rule{},
				Chain: rules.Chain{
					Table:       "some-table",
					ParentChain: "INPUT",
					Prefix:      "some-prefix",
				},
			}
			fakePlanner.GetRulesReturns(rulesWithChain, nil)
		})

		It("enforces rules on configured interval", func() {
			signals := make(chan os.Signal)
			ready := make(chan struct{})
			go p.Run(signals, ready)
			Eventually(ready).Should(BeClosed())
			Eventually(fakePlanner.GetRulesCallCount()).Should(BeNumerically(">", 0))
			Eventually(fakeEnforcer.EnforceRulesAndChainCallCount()).Should(BeNumerically(">", 0))
			signals <- os.Interrupt

			rws := fakeEnforcer.EnforceRulesAndChainArgsForCall(0)
			Expect(rws).To(Equal(rulesWithChain))
		})
		It("emits time metrics", func() {
			signals := make(chan os.Signal)
			ready := make(chan struct{})
			go p.Run(signals, ready)
			Eventually(ready).Should(BeClosed())
			Eventually(timeMetricsEmitter.EmitAllCallCount).Should(BeNumerically(">", 0))
			signals <- os.Interrupt
		})

		Context("when planner errors", func() {
			BeforeEach(func() {
				fakePlanner.GetRulesReturns(rulesWithChain, errors.New("eggplant"))
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
				fakeEnforcer.EnforceRulesAndChainReturns(errors.New("eggplant"))
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
