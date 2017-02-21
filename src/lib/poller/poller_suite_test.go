package poller_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPoller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Poller Suite")
}
