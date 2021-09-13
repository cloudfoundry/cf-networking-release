package uaa_client_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUaaClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UaaClient Suite")
}
