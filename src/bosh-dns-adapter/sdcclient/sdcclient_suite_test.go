package sdcclient_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSdcclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sdcclient Suite")
}
