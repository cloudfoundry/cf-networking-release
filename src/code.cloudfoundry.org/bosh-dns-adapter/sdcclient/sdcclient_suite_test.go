package sdcclient_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSdcclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sdcclient Suite")
}
