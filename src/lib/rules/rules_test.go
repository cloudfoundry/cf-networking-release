package rules_test

import (
	"lib/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Rules", func() {
	Describe("AppendComment", func() {
		var originalRule rules.IPTablesRule
		BeforeEach(func() {
			originalRule = rules.IPTablesRule{"some", "rule"}
		})
		It("appends the comment to the iptables rule, replacing spaces with underscores", func() {
			rule := rules.AppendComment(originalRule, `some:comment statement`)
			Expect(rule).To(Equal(rules.IPTablesRule{
				"some", "rule", "-m", "comment", "--comment", `some:comment_statement`,
			}))
		})
	})

	Describe("NewLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewLogRule([]string{}, "some-very-very-very-long-app-guid")
				Expect(rule).To(ContainElement(`"some-very-very-very-long-app "`))
			})
		})
	})

	Describe("NewMarkAllowLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewMarkAllowLogRule("", "", 0, "", "some-very-very-very-long-app-guid")
				Expect(rule).To(ContainElement(`"OK__some-very-very-very-long "`))
			})
		})
	})

	Describe("NewNetOutDefaultLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewNetOutDefaultLogRule("some-very-very-very-long-app-guid")
				Expect(rule).To(ContainElement(`all`))
				Expect(rule).To(ContainElement(`"OK_some-very-very-very-long- "`))
			})
		})
	})

	Describe("NewOverlayDefaultRejectLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewOverlayDefaultRejectLogRule("some-very-very-very-long-app-guid", "", 5)
				Expect(rule).To(gomegamatchers.ContainSequence(rules.IPTablesRule{
					"-m", "limit", "--limit", "5/s", "--limit-burst", "5",
				}))
				Expect(rule).To(ContainElement(`"DENY_C2C_some-very-very-very "`))
			})
		})
	})

	Describe("NewNetOutDefaultRejectLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewNetOutDefaultRejectLogRule("some-very-very-very-long-app-guid", 3)
				Expect(rule).To(gomegamatchers.ContainSequence(rules.IPTablesRule{
					"-m", "limit", "--limit", "3/s", "--limit-burst", "3",
				}))
				Expect(rule).To(ContainElement(`"DENY_some-very-very-very-lon "`))
			})
		})
	})
})
