package rule_updater_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRuleUpdater(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RuleUpdater Suite")
}
