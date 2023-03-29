package api_v0_internal_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApiV0Internal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ApiV0Internal Suite")
}
