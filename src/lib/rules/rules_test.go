package rules_test

import (
	"errors"
	"lib/fakes"
	"lib/rules"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Rules", func() {
	var (
		logger   *lagertest.TestLogger
		iptables *fakes.IPTables
		restorer *fakes.Restorer
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		iptables = &fakes.IPTables{}
		restorer = &fakes.Restorer{}
	})

	Describe("GenericRule", func() {
		var rule rules.GenericRule

		Describe("Enforce", func() {
			BeforeEach(func() {
				rule = rules.GenericRule{
					Properties: []string{"-j", "some-other-chain"},
				}
			})

			It("appends an iptables rule to the chain supplied", func() {
				err := rule.Enforce("some-table", "some-chain", iptables, logger)
				Expect(err).NotTo(HaveOccurred())

				Expect(iptables.AppendUniqueCallCount()).To(Equal(1))
				table, chain, ruleSpec := iptables.AppendUniqueArgsForCall(0)
				Expect(table).To(Equal("some-table"))
				Expect(chain).To(Equal("some-chain"))
				Expect(ruleSpec).To(Equal([]string{"-j", "some-other-chain"}))
				Expect(logger).To(gbytes.Say(`enforce-rule.*{"chain":"some-chain","properties":"\[-j some-other-chain\]","table":"some-table"}`))
			})

			Context("when theres an error appending the rule", func() {
				It("logs and returns a useful error", func() {
					iptables.AppendUniqueReturns(errors.New("raspberry"))

					err := rule.Enforce("some-table", "some-chain", iptables, logger)
					Expect(err).To(MatchError("appending rule: raspberry"))
					Expect(logger).To(gbytes.Say("append-rule.*raspberry"))
				})
			})
		})
	})
	Describe("RuleSet", func() {
		var ruleSet rules.RuleSet

		Describe("BulkEnforce", func() {
			BeforeEach(func() {
				ruleSet = rules.RuleSet{
					Rules: []rules.GenericRule{
						rules.NewMarkSetRule("1.2.3.4", "A", "a-guid"),
						rules.NewMarkSetRule("2.2.2.2", "B", "b-guid"),
					},
				}
			})
			It("aggregates the rules and enforces them", func() {
				err := ruleSet.BulkAppend("some-table", "some-chain", restorer, logger)
				Expect(err).NotTo(HaveOccurred())

				Expect(restorer.RestoreCallCount()).To(Equal(1))
				restoreInput := restorer.RestoreArgsForCall(0)
				Expect(restoreInput).To(ContainSubstring("*some-table\n"))
				Expect(restoreInput).To(ContainSubstring("-A some-chain --source 1.2.3.4 --jump MARK --set-xmark 0xA -m comment --comment src:a-guid\n"))
				Expect(restoreInput).To(ContainSubstring("-A some-chain --source 2.2.2.2 --jump MARK --set-xmark 0xB -m comment --comment src:b-guid\n"))
				Expect(restoreInput).To(ContainSubstring("COMMIT\n"))
			})
			Context("when the restorer fails", func() {
				It("logs and returns a useful error", func() {
					restorer.RestoreReturns(errors.New("banana"))

					err := ruleSet.BulkAppend("some-table", "some-chain", restorer, logger)

					Expect(err).To(MatchError("bulk appending rules: banana"))
					Expect(logger).To(gbytes.Say("bulk-append.*banana"))
				})
			})
		})
	})
})
