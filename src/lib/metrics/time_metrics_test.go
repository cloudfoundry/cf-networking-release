package metrics_test

import (
	"errors"
	"lib/metrics"
	"time"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("TimeMetrics", func() {

	var (
		timeMetrics *metrics.MetricsSender
		logger      *lagertest.TestLogger
	)

	getValueMetrics := func() []*events.ValueMetric {
		emittedMetrics := []*events.ValueMetric{}
		for _, message := range fakeDropsonde.GetMessages() {
			emittedMetrics = append(emittedMetrics, message.Event.(*events.ValueMetric))
		}
		return emittedMetrics
	}

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		timeMetrics = &metrics.MetricsSender{
			Logger: logger,
		}
		fakeDropsonde.Reset()
	})

	Describe("SendDuration", func() {
		var (
			name     string
			duration time.Duration
		)
		BeforeEach(func() {
			name = "name"
			duration = 5 * time.Second
		})
		It("sends a value through dropsonde", func() {
			timeMetrics.SendDuration(name, duration)

			Eventually(fakeDropsonde.GetMessages).Should(HaveLen(1))
			Eventually(getValueMetrics).Should(ConsistOf(
				[]*events.ValueMetric{
					&events.ValueMetric{
						Name:  proto.String("name"),
						Unit:  proto.String("ms"),
						Value: proto.Float64(5 * time.Second.Seconds() * 1000),
					},
				},
			))
		})

		Context("when dropsonde returns an error", func() {
			BeforeEach(func() {
				fakeDropsonde.ReturnError = errors.New("banana")
			})
			It("logs the error from dropsonde", func() {
				timeMetrics.SendDuration(name, duration)

				Expect(logger).To(gbytes.Say("sending-metric.*banana"))
			})
		})
	})
})
