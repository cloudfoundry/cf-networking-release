package rules_test

import (
	"errors"
	"netman-agent/fakes"
	"netman-agent/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Rules", func() {
	var (
		logger   *lagertest.TestLogger
		iptables *fakes.IPTables
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		iptables = &fakes.IPTables{}
	})

	Describe("LocalAllowRule", func() {
		var rule rules.LocalAllowRule

		Describe("Enforce", func() {
			BeforeEach(func() {
				rule = rules.LocalAllowRule{
					SrcIP:    "1.2.3.4",
					DstIP:    "5.6.7.8",
					Port:     8080,
					Proto:    "tcp",
					IPTables: iptables,
					Logger:   logger,
				}
			})

			It("appends an iptables rule to the chain supplied", func() {
				err := rule.Enforce("some-chain")
				Expect(err).NotTo(HaveOccurred())

				Expect(iptables.AppendUniqueCallCount()).To(Equal(1))
				table, chain, ruleSpec := iptables.AppendUniqueArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("some-chain"))
				Expect(ruleSpec).To(Equal([]string{
					"-i", "cni-flannel0",
					"-s", "1.2.3.4",
					"-d", "5.6.7.8",
					"-p", "tcp",
					"--dport", "8080",
					"-j", "ACCEPT",
				}))

				Expect(logger).To(gbytes.Say(`enforce-local-rule.*{"dstIP":"5.6.7.8","port":8080,"proto":"tcp","srcIP":"1.2.3.4"}`))
			})

			Context("when theres an error appending the rule", func() {
				It("logs and returns a useful error", func() {
					iptables.AppendUniqueReturns(errors.New("raspberry"))

					err := rule.Enforce("some-chain")
					Expect(err).To(MatchError("appending rule: raspberry"))
					Expect(logger).To(gbytes.Say("append-rule.*raspberry"))
				})
			})
		})
	})

	Describe("RemoteAllowRule", func() {
		var rule rules.RemoteAllowRule

		Describe("Enforce", func() {
			BeforeEach(func() {
				rule = rules.RemoteAllowRule{
					SrcTag:   "BEEF",
					DstIP:    "5.6.7.8",
					Port:     8080,
					Proto:    "tcp",
					VNI:      42,
					IPTables: iptables,
					Logger:   logger,
				}
			})

			It("appends an iptables rule to the chain supplied", func() {
				err := rule.Enforce("some-chain")
				Expect(err).NotTo(HaveOccurred())

				Expect(iptables.AppendUniqueCallCount()).To(Equal(1))
				table, chain, ruleSpec := iptables.AppendUniqueArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("some-chain"))
				Expect(ruleSpec).To(Equal([]string{
					"-i", "flannel.42",
					"-d", "5.6.7.8",
					"-p", "tcp",
					"--dport", "8080",
					"-m", "mark", "--mark", "0xBEEF",
					"-j", "ACCEPT",
				}))

				Expect(logger).To(gbytes.Say(`enforce-remote-rule.*{"dstIP":"5.6.7.8","port":8080,"proto":"tcp","srcTag":"BEEF","vni":42}`))
			})

			Context("when theres an error appending the rule", func() {
				It("logs and returns a useful error", func() {
					iptables.AppendUniqueReturns(errors.New("raspberry"))

					err := rule.Enforce("some-chain")
					Expect(err).To(MatchError("appending rule: raspberry"))
					Expect(logger).To(gbytes.Say("append-rule.*raspberry"))
				})
			})
		})
	})

	Describe("LocalTagRule", func() {
		var rule rules.LocalTagRule

		Describe("Enforce", func() {
			BeforeEach(func() {
				rule = rules.LocalTagRule{
					SourceTag:         "BEEF",
					SourceContainerIP: "5.6.7.8",
					IPTables:          iptables,
					Logger:            logger,
				}
			})

			It("appends an iptables rule to the chain supplied", func() {
				err := rule.Enforce("some-chain")
				Expect(err).NotTo(HaveOccurred())

				Expect(iptables.AppendUniqueCallCount()).To(Equal(1))
				table, chain, ruleSpec := iptables.AppendUniqueArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("some-chain"))
				Expect(ruleSpec).To(Equal([]string{
					"-s", "5.6.7.8",
					"-j", "MARK", "--set-xmark", "0xBEEF",
				}))

				Expect(logger).To(gbytes.Say(`set-local-tag.*{"srcIP":"5.6.7.8","srcTag":"BEEF"}`))
			})

			Context("when theres an error appending the rule", func() {
				It("logs and returns a useful error", func() {
					iptables.AppendUniqueReturns(errors.New("raspberry"))

					err := rule.Enforce("some-chain")
					Expect(err).To(MatchError("appending rule: raspberry"))
					Expect(logger).To(gbytes.Say("append-rule.*raspberry"))
				})
			})
		})
	})
})
