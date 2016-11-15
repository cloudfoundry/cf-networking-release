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
})
