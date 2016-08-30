package port_allocator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPortAllocator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PortAllocator Suite")
}
