package api_0_internal_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApi0Internal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api0Internal Suite")
}
