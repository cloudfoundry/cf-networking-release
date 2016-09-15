package server_metrics_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestServerMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServerMetrics Suite")
}
