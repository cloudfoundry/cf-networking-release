package addresstable_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAddresstable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Addresstable Suite")
}
