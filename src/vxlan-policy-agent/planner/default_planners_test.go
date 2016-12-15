package planner_test

import (
	"lib/rules"
	"vxlan-policy-agent/enforcer"
	"vxlan-policy-agent/fakes"
	"vxlan-policy-agent/planner"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Local planner", func() {
	var (
		localPlanner       *planner.VxlanDefaultLocalPlanner
		logger             *lagertest.TestLogger
		chain              enforcer.Chain
		loggingStateGetter *fakes.LoggingStateGetter
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		loggingStateGetter = &fakes.LoggingStateGetter{}
		chain = enforcer.Chain{
			Table:       "some-table",
			ParentChain: "INPUT",
			Prefix:      "some-prefix",
		}
		localPlanner = &planner.VxlanDefaultLocalPlanner{
			Logger:       logger,
			LocalSubnet:  "some-subnet",
			Chain:        chain,
			LoggingState: loggingStateGetter,
		}
	})

	Describe("GetRules", func() {
		Context("when iptables logging is disabled", func() {
			BeforeEach(func() {
				loggingStateGetter.IsEnabledReturns(false)
			})
			It("returns a ruleset with accept-existing and default-deny rules", func() {
				ruleSet, err := localPlanner.GetRules()
				Expect(err).NotTo(HaveOccurred())
				Expect(ruleSet).To(HaveLen(2))
				Expect(ruleSet[0]).To(Equal(rules.NewAcceptExistingLocalRule()))
				Expect(ruleSet[1]).To(Equal(rules.NewDefaultDenyLocalRule("some-subnet")))
				Expect(loggingStateGetter.IsEnabledCallCount()).To(Equal(1))
			})
		})

		Context("when iptables logging is enabled", func() {
			BeforeEach(func() {
				loggingStateGetter.IsEnabledReturns(true)
			})
			It("includes a logging rule immediately before the default-deny", func() {
				ruleSet, err := localPlanner.GetRules()
				Expect(err).NotTo(HaveOccurred())
				Expect(ruleSet).To(HaveLen(3))
				Expect(ruleSet[0]).To(Equal(rules.NewAcceptExistingLocalRule()))
				Expect(ruleSet[1]).To(Equal(rules.NewLogLocalRejectRule("some-subnet")))
				Expect(ruleSet[2]).To(Equal(rules.NewDefaultDenyLocalRule("some-subnet")))
				Expect(loggingStateGetter.IsEnabledCallCount()).To(Equal(1))
			})
		})
	})
})

var _ = Describe("Remote planner", func() {
	var (
		remotePlanner      *planner.VxlanDefaultRemotePlanner
		logger             *lagertest.TestLogger
		chain              enforcer.Chain
		loggingStateGetter *fakes.LoggingStateGetter
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		loggingStateGetter = &fakes.LoggingStateGetter{}
		chain = enforcer.Chain{
			Table:       "some-table",
			ParentChain: "INPUT",
			Prefix:      "some-prefix",
		}
		remotePlanner = &planner.VxlanDefaultRemotePlanner{
			Logger:       logger,
			VNI:          42,
			Chain:        chain,
			LoggingState: loggingStateGetter,
		}
	})

	Describe("GetRules", func() {
		Context("when iptables logging is disabled", func() {
			BeforeEach(func() {
				loggingStateGetter.IsEnabledReturns(false)
			})
			It("returns a ruleset with accept-existing and default-deny rules", func() {
				ruleSet, err := remotePlanner.GetRules()
				Expect(err).NotTo(HaveOccurred())
				Expect(ruleSet).To(HaveLen(2))
				Expect(ruleSet[0]).To(Equal(rules.NewAcceptExistingRemoteRule(42)))
				Expect(ruleSet[1]).To(Equal(rules.NewDefaultDenyRemoteRule(42)))
				Expect(loggingStateGetter.IsEnabledCallCount()).To(Equal(1))
			})
		})

		Context("when iptables logging is enabled", func() {
			BeforeEach(func() {
				loggingStateGetter.IsEnabledReturns(true)
			})
			It("includes a logging rule immediately before the default-deny", func() {
				ruleSet, err := remotePlanner.GetRules()
				Expect(err).NotTo(HaveOccurred())
				Expect(ruleSet).To(HaveLen(3))
				Expect(ruleSet[0]).To(Equal(rules.NewAcceptExistingRemoteRule(42)))
				Expect(ruleSet[1]).To(Equal(rules.NewLogRemoteRejectRule(42)))
				Expect(ruleSet[2]).To(Equal(rules.NewDefaultDenyRemoteRule(42)))
				Expect(loggingStateGetter.IsEnabledCallCount()).To(Equal(1))
			})
		})
	})
})
