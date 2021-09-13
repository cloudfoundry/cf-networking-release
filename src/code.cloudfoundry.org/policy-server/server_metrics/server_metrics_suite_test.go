package server_metrics_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServerMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServerMetrics Suite")
}
