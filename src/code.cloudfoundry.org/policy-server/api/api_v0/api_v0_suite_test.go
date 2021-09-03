package api_v0_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApiV0(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ApiV0 Suite")
}
