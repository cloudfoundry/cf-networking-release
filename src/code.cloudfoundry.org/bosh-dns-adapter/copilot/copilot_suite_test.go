package copilot_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCopilot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Copilot Suite")
}
