package flannel_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFlannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flannel Suite")
}
