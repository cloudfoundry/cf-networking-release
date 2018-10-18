package vip_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVip(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vip Suite")
}
