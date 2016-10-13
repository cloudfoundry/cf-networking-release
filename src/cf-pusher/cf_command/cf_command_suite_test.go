package cf_command_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCfCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CfCommand Suite")
}
