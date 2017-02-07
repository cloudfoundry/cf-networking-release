package metrics_test

import (
	"errors"
	"lib/metrics"
	"os"
	"time"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
)

const (
	interval = 100 * time.Millisecond
)

var _ = Describe("MetricsEmitter", func() {
	var (
		metricsEmitter     *metrics.MetricsEmitter
		metricsEmitterProc ifrit.Process
		logger             *lagertest.TestLogger

		fakeSource  metrics.MetricSource
		fakeSource2 metrics.MetricSource
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		fakeDropsonde.Reset()
		fakeSource = metrics.MetricSource{
			Name: "fakeSource",
			Unit: "fakeUnit",
			Getter: func() (float64, error) {
				return 42, nil
			},
		}
		fakeSource2 = metrics.MetricSource{
			Name: "fakeSource2",
			Unit: "fakeUnit",
			Getter: func() (float64, error) {
				return 42, nil
			},
		}
	})

	AfterEach(func() {
		metricsEmitterProc.Signal(os.Interrupt)
		Eventually(metricsEmitterProc.Wait()).Should(Receive())
	})

	It("immediately does one round of metrics reporting", func() {
		metricsEmitter = metrics.NewMetricsEmitter(logger, interval, fakeSource, fakeSource2)
		metricsEmitterProc = ifrit.Invoke(metricsEmitter)
		Eventually(metricsEmitterProc.Ready()).Should(BeClosed())
		Expect(fakeDropsonde.GetMessages()).To(HaveLen(2))

		metric := fakeDropsonde.GetMessages()[0].Event.(*events.ValueMetric)
		Expect(metric.Name).To(Equal(proto.String("fakeSource")))
		Expect(metric.Unit).To(Equal(proto.String("fakeUnit")))
		Expect(*metric.Value).To(Equal(42.0))

		metric = fakeDropsonde.GetMessages()[1].Event.(*events.ValueMetric)
		Expect(metric.Name).To(Equal(proto.String("fakeSource2")))
		Expect(metric.Unit).To(Equal(proto.String("fakeUnit")))
		Expect(*metric.Value).To(Equal(42.0))
	})

	It("reports all metrics", func() {
		metricsEmitter = metrics.NewMetricsEmitter(logger, interval, fakeSource, fakeSource2)
		metricsEmitterProc = ifrit.Invoke(metricsEmitter)
		Eventually(fakeDropsonde.GetMessages).Should(HaveLen(4))

		metric := fakeDropsonde.GetMessages()[2].Event.(*events.ValueMetric)
		Expect(metric.Name).To(Equal(proto.String("fakeSource")))
		Expect(metric.Unit).To(Equal(proto.String("fakeUnit")))
		Expect(*metric.Value).To(Equal(42.0))

		metric = fakeDropsonde.GetMessages()[3].Event.(*events.ValueMetric)
		Expect(metric.Name).To(Equal(proto.String("fakeSource2")))
		Expect(metric.Unit).To(Equal(proto.String("fakeUnit")))
		Expect(*metric.Value).To(Equal(42.0))
	})

	Context("when the metric source getter fails", func() {
		BeforeEach(func() {
			badSource := metrics.MetricSource{
				Name:   "badSource",
				Unit:   "whatevs",
				Getter: func() (float64, error) { return 1, errors.New("potato") },
			}
			metricsEmitter = metrics.NewMetricsEmitter(logger, interval, badSource)
		})

		It("logs the error", func() {
			metricsEmitterProc = ifrit.Invoke(metricsEmitter)
			Eventually(logger).Should(gbytes.Say("metric-getter.*potato.*badSource"))
		})

		It("does not send a value", func() {
			metricsEmitterProc = ifrit.Invoke(metricsEmitter)
			Consistently(fakeDropsonde.GetMessages, "1s").Should(BeEmpty())
		})
	})
})
