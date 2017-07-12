package merger_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMerger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Merger Suite")
}
