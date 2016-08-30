package bindmount_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBindmount(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bindmount Suite")
}
