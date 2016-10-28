package poller_test

import (
	"errors"
	"lib/rules"
	"vxlan-policy-agent/fakes"
	"vxlan-policy-agent/poller"

	libfakes "lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Single Poll Cycle", func() {
	Describe("Run", func() {
		var (
			p                  *poller.SinglePollCycle
			fakePlanner        *fakes.Planner
			fakeEnforcer       *libfakes.RuleEnforcer
			timeMetricsEmitter *fakes.TimeMetricsEmitter
			rulesWithChain     rules.RulesWithChain
		)

		BeforeEach(func() {
			fakePlanner = &fakes.Planner{}
			fakeEnforcer = &libfakes.RuleEnforcer{}
			timeMetricsEmitter = &fakes.TimeMetricsEmitter{}

			p = &poller.SinglePollCycle{
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
			err := p.DoCycle()
			Expect(err).NotTo(HaveOccurred())
			Expect(fakePlanner.GetRulesCallCount()).To(Equal(1))
			Expect(fakeEnforcer.EnforceRulesAndChainCallCount()).To(Equal(1))

			rws := fakeEnforcer.EnforceRulesAndChainArgsForCall(0)
			Expect(rws).To(Equal(rulesWithChain))
		})

		It("emits time metrics", func() {
			err := p.DoCycle()
			Expect(err).NotTo(HaveOccurred())
			Expect(timeMetricsEmitter.EmitAllCallCount()).To(Equal(1))
		})

		Context("when planner errors", func() {
			BeforeEach(func() {
				fakePlanner.GetRulesReturns(rulesWithChain, errors.New("eggplant"))
			})

			It("logs the error and returns", func() {
				err := p.DoCycle()
				Expect(err).To(MatchError("get-rules: eggplant"))

				Expect(fakeEnforcer.EnforceOnChainCallCount()).To(Equal(0))
				Expect(timeMetricsEmitter.EmitAllCallCount()).To(Equal(0))
			})
		})

		Context("when enforcer errors", func() {
			BeforeEach(func() {
				fakeEnforcer.EnforceRulesAndChainReturns(errors.New("eggplant"))
			})

			It("logs the error and returns", func() {
				err := p.DoCycle()
				Expect(err).To(MatchError("enforce: eggplant"))

				Expect(timeMetricsEmitter.EmitAllCallCount()).To(Equal(0))
			})
		})
	})
})
