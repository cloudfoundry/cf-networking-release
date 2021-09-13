package policy_client_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPolicyClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PolicyClient Suite")
}
