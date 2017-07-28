package api_v0_internal_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApiV0Internal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ApiV0Internal Suite")
}
