package port_allocator_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPortAllocator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PortAllocator Suite")
}
