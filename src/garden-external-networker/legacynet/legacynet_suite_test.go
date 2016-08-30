package legacynet_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLegacynet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Legacynet Suite")
}
