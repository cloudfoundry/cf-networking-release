package metrics_test

import (
	"lib/metrics"
	"time"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TimeMetrics", func() {

	var (
		timeMetrics *metrics.TimeMetrics
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("test")
		timeMetrics = &metrics.TimeMetrics{
			Logger: logger,
		}
		fakeDropsonde.Reset()
	})

	Describe("EmitAll", func() {
		It("sends a value for each duration", func() {
			durations := map[string]time.Duration{
				"one": time.Second,
				"two": time.Hour,
			}

			timeMetrics.EmitAll(durations)

			Eventually(fakeDropsonde.GetMessages).Should(HaveLen(2))

			getValueMetrics := func() []*events.ValueMetric {
				emittedMetrics := []*events.ValueMetric{}
				for _, message := range fakeDropsonde.GetMessages() {
					emittedMetrics = append(emittedMetrics, message.Event.(*events.ValueMetric))
				}
				return emittedMetrics
			}

			Eventually(getValueMetrics).Should(ConsistOf(
				[]*events.ValueMetric{
					&events.ValueMetric{
						Name:  proto.String("one"),
						Unit:  proto.String("ms"),
						Value: proto.Float64(time.Second.Seconds() * 1000),
					},
					&events.ValueMetric{
						Name:  proto.String("two"),
						Unit:  proto.String("ms"),
						Value: proto.Float64(time.Hour.Seconds() * 1000),
					},
				},
			))
		})
	})
})
