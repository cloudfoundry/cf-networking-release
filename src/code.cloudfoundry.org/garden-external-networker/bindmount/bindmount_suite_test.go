package bindmount_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBindmount(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bindmount Suite")
}
