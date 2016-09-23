package testsupport_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTestsupport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testsupport Suite")
}
