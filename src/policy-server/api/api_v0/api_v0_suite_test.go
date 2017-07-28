package api_v0_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApiV0(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ApiV0 Suite")
}
