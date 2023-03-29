package cf_command_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCfCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CfCommand Suite")
}
