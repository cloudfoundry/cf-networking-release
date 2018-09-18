package psclient_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPsclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Psclient Suite")
}
