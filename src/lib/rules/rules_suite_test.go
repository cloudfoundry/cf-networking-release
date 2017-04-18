package rules_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rules Suite")
}

var _ = AfterSuite(func() {
	iptablesCmd := exec.Command("iptables", "-F", "FORWARD")
	Expect(iptablesCmd.Run()).To(Succeed())
})
