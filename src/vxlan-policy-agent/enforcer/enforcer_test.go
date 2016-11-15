package enforcer_test

import (
	"errors"
	"lib/rules"
	"vxlan-policy-agent/enforcer"
	"vxlan-policy-agent/fakes"

	libfakes "lib/fakes"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Enforcer", func() {
	Describe("Enforce", func() {
		var (
			fakeRule     rules.GenericRule
			fakeRule2    rules.GenericRule
			iptables     *libfakes.IPTablesExtended
			timestamper  *fakes.TimeStamper
			logger       *lagertest.TestLogger
			ruleEnforcer *enforcer.Enforcer
		)

		BeforeEach(func() {
			fakeRule = rules.GenericRule{
				Properties: []string{"rule1"},
			}
			fakeRule2 = rules.GenericRule{
				Properties: []string{"rule2"},
			}

			timestamper = &fakes.TimeStamper{}
			logger = lagertest.NewTestLogger("test")
			iptables = &libfakes.IPTablesExtended{}

			timestamper.CurrentTimeReturns(42)
			ruleEnforcer = enforcer.NewEnforcer(logger, timestamper, iptables)
		})

		It("enforces all the rules it receives on the correct chain", func() {
			rulesToAppend := []rules.GenericRule{fakeRule, fakeRule2}
			err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", rulesToAppend...)
			Expect(err).NotTo(HaveOccurred())

			Expect(iptables.BulkAppendCallCount()).To(Equal(1))
			tbl, chain, rules := iptables.BulkAppendArgsForCall(0)
			Expect(tbl).To(Equal("some-table"))
			Expect(chain).To(Equal("foo42"))
			Expect(rules).To(Equal(rulesToAppend))
		})

		Context("when the bulk append fails", func() {
			BeforeEach(func() {
				iptables.BulkAppendReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				rulesToAppend := []rules.GenericRule{fakeRule, fakeRule2}
				err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", rulesToAppend...)
				Expect(err).To(MatchError("bulk appending: banana"))
			})
		})

		It("creates a timestamped chain", func() {
			err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", []rules.GenericRule{fakeRule}...)
			Expect(err).NotTo(HaveOccurred())

			Expect(iptables.NewChainCallCount()).To(Equal(1))
			tableName, chainName := iptables.NewChainArgsForCall(0)
			Expect(tableName).To(Equal("some-table"))
			Expect(chainName).To(Equal("foo42"))
		})

		It("inserts the new chain into the chain", func() {
			err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", []rules.GenericRule{fakeRule}...)
			Expect(err).NotTo(HaveOccurred())

			Expect(iptables.InsertCallCount()).To(Equal(1))
			tableName, chainName, pos, ruleSpec := iptables.InsertArgsForCall(0)
			Expect(tableName).To(Equal("some-table"))
			Expect(chainName).To(Equal("some-chain"))
			Expect(pos).To(Equal(1))
			Expect(ruleSpec).To(Equal([]string{"-j", "foo42"}))
		})

		Context("when there is an older timestamped chain", func() {
			BeforeEach(func() {
				iptables.ListReturns([]string{
					"-A some-chain -j foo0000000001",
					"-A some-chain -j foo9999999999",
				}, nil)
			})
			It("gets deleted", func() {
				err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", []rules.GenericRule{fakeRule}...)
				Expect(err).NotTo(HaveOccurred())

				Expect(iptables.DeleteCallCount()).To(Equal(1))
				table, chain, ruleSpec := iptables.DeleteArgsForCall(0)
				Expect(table).To(Equal("some-table"))
				Expect(chain).To(Equal("some-chain"))
				Expect(ruleSpec).To(Equal([]string{"-j", "foo0000000001"}))
				Expect(iptables.ClearChainCallCount()).To(Equal(1))
				table, chain = iptables.ClearChainArgsForCall(0)
				Expect(table).To(Equal("some-table"))
				Expect(chain).To(Equal("foo0000000001"))
				Expect(iptables.DeleteChainCallCount()).To(Equal(1))
				table, chain = iptables.DeleteChainArgsForCall(0)
				Expect(table).To(Equal("some-table"))
				Expect(chain).To(Equal("foo0000000001"))
			})
		})

		Context("when inserting the new chain fails", func() {
			BeforeEach(func() {
				iptables.InsertReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", []rules.GenericRule{fakeRule}...)
				Expect(err).To(MatchError("inserting chain: banana"))

				Expect(logger).To(gbytes.Say("insert-chain.*banana"))
			})
		})

		Context("when there are errors cleaning up old rules", func() {
			BeforeEach(func() {
				iptables.ListReturns(nil, errors.New("blueberry"))
			})

			It("it logs and returns a useful error", func() {
				err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", []rules.GenericRule{fakeRule}...)
				Expect(err).To(MatchError("listing forward rules: blueberry"))

				Expect(logger).To(gbytes.Say("cleanup-rules.*blueberry"))
			})
		})

		Context("when there are errors cleaning up old chains", func() {
			BeforeEach(func() {
				iptables.DeleteReturns(errors.New("banana"))
				iptables.ListReturns([]string{"-A some-chain -j foo0000000001"}, nil)
			})

			It("returns a useful error", func() {
				err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", []rules.GenericRule{fakeRule}...)
				Expect(err).To(MatchError("cleanup old chain: banana"))
			})
		})

		Context("when creating the new chain fails", func() {
			BeforeEach(func() {
				iptables.NewChainReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				err := ruleEnforcer.Enforce("some-table", "some-chain", "foo", []rules.GenericRule{fakeRule}...)
				Expect(err).To(MatchError("creating chain: banana"))

				Expect(logger).To(gbytes.Say("create-chain.*banana"))
			})
		})
	})
})
