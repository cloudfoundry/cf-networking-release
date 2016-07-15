package styles_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStyles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Styles Suite")
}
