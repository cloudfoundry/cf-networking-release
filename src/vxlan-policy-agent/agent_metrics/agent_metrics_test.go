package agent_metrics_test

import (
	"netmon/integration/fakes"
	"time"
	"vxlan-policy-agent/agent_metrics"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/cloudfoundry/dropsonde"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TimeMetrics", func() {

	var (
		timeMetrics *agent_metrics.TimeMetrics
		fakeMetron  fakes.FakeMetron
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("test")
		timeMetrics = &agent_metrics.TimeMetrics{
			Logger: logger,
		}
		fakeMetron = fakes.New()
		metronAddress := fakeMetron.Address()
		dropsonde.Initialize(metronAddress, "whatever")
	})

	AfterEach(func() {
		Expect(fakeMetron.Close()).To(Succeed())
	})

	Describe("EmitAll", func() {
		It("sends a value for each duration", func() {
			durations := map[string]time.Duration{
				"one": time.Second,
				"two": time.Hour,
			}

			timeMetrics.EmitAll(durations)

			Eventually(fakeMetron.AllEvents).Should(ContainElement(fakes.Event{
				EventType: "ValueMetric",
				Name:      "one",
				Origin:    "whatever",
				Value:     time.Second.Seconds() * 1000,
			}))
			Eventually(fakeMetron.AllEvents).Should(ContainElement(fakes.Event{
				EventType: "ValueMetric",
				Name:      "two",
				Origin:    "whatever",
				Value:     time.Hour.Seconds() * 1000,
			}))
		})
	})
})
