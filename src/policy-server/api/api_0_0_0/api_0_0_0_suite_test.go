package api_0_0_0_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApi000(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api000 Suite")
}
