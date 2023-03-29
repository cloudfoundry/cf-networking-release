package cc_client_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCcClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CcClient Suite")
}
