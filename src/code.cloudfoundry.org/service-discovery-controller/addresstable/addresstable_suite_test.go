package addresstable_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAddresstable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Addresstable Suite")
}
