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

var _ = Describe("Metrics Sender", func() {

	var (
		metricsSender *metrics.MetricsSender
		logger        *lagertest.TestLogger
	)

	getValueMetrics := func() []*events.ValueMetric {
		emittedMetrics := []*events.ValueMetric{}
		for _, message := range fakeDropsonde.GetMessages() {
			emittedMetrics = append(emittedMetrics, message.Event.(*events.ValueMetric))
		}
		return emittedMetrics
	}

	getCounterEvents := func() []*events.CounterEvent {
		emittedEvents := []*events.CounterEvent{}
		for _, message := range fakeDropsonde.GetMessages() {
			emittedEvents = append(emittedEvents, message.Event.(*events.CounterEvent))
		}
		return emittedEvents
	}

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		metricsSender = &metrics.MetricsSender{
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
			metricsSender.SendDuration(name, duration)

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
				metricsSender.SendDuration(name, duration)

				Expect(logger).To(gbytes.Say("sending-metric.*banana"))
			})
		})
	})

	Describe("IncrementCounter", func() {
		It("sends a value through dropsonde", func() {
			metricsSender.IncrementCounter("foo")
			Eventually(fakeDropsonde.GetMessages).Should(HaveLen(1))
			Eventually(getCounterEvents).Should(ConsistOf(
				[]*events.CounterEvent{
					&events.CounterEvent{
						Name:  proto.String("foo"),
						Delta: proto.Uint64(1),
					},
				},
			))
		})

		Context("when dropsonde returns an error", func() {
			BeforeEach(func() {
				fakeDropsonde.ReturnError = errors.New("banana")
			})
			It("logs the error from dropsonde", func() {
				metricsSender.IncrementCounter("foo")

				Expect(logger).To(gbytes.Say("sending-metric.*banana"))
			})
		})
	})
})
