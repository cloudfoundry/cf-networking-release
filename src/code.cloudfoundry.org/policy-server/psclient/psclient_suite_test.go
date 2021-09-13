package psclient_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPsclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Psclient Suite")
}
