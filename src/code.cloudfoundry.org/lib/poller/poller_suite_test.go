package poller_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPoller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Poller Suite")
}
