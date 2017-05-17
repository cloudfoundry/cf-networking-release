package rules_test

import (
	"lib/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

	Describe("NewMarkLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewMarkLogRule("", "", 0, "", "some-very-very-very-long-app-guid")
				Expect(rule).To(ContainElement(`"OK__some-very-very-very-long "`))
			})
		})
	})

	Describe("NewNetOutDefaultLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewNetOutDefaultLogRule("some-very-very-very-long-app-guid")
				Expect(rule).To(ContainElement(`"OK_some-very-very-very-long- "`))
			})
		})
	})

	Describe("NewOverlayDefaultRejectLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewOverlayDefaultRejectLogRule("some-very-very-very-long-app-guid", "")
				Expect(rule).To(ContainElement(`"DENY_C2C_some-very-very-very "`))
			})
		})
	})

	Describe("NewNetOutDefaultRejectLogRule", func() {
		Context("when the log prefix is greater than 28 characters", func() {
			It("shortens the log-prefix to 28 characters and adds a space", func() {
				rule := rules.NewNetOutDefaultRejectLogRule("some-very-very-very-long-app-guid")
				Expect(rule).To(ContainElement(`"DENY_some-very-very-very-lon "`))
			})
		})
	})
})
