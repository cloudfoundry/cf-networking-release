package planner_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPolicyClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Planner Suite")
}
