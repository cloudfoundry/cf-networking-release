package metrics_test

import (
	"github.com/cloudfoundry/dropsonde/emitter/fake"
	"github.com/cloudfoundry/dropsonde/metric_sender"
	"github.com/cloudfoundry/dropsonde/metrics"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var fakeEventEmitter *fake.FakeEventEmitter

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

var _ = BeforeSuite(func() {
	fakeEventEmitter = fake.NewFakeEventEmitter("MetricsTest")
	sender := metric_sender.NewMetricSender(fakeEventEmitter)
	metrics.Initialize(sender, nil)
})

var _ = AfterSuite(func() {
	fakeEventEmitter.Close()
})
