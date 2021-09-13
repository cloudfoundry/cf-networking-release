package rules_test

import (
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rules Suite")
}

var _ = SynchronizedAfterSuite(func() {
	// runs once on every parallel node
}, func() {
	// runs only on node #1 after all parallel ones are finished
	iptablesCmd := exec.Command("iptables", "-F", "FORWARD")
	Expect(iptablesCmd.Run()).To(Succeed())
})
