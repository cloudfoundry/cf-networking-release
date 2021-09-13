package cni_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCni(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CNI Suite")
}
