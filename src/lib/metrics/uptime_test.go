package metrics_test

import (
	"lib/metrics"
	"os"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

const (
	interval = 100 * time.Millisecond
)

var _ = Describe("Uptime", func() {
	var (
		uptime     *metrics.Uptime
		uptimeProc ifrit.Process
	)

	BeforeEach(func() {
		fakeEventEmitter.Reset()
		uptime = metrics.NewUptime(interval)
		uptimeProc = ifrit.Invoke(uptime)
	})

	Context("stops automatically", func() {

		AfterEach(func() {
			uptimeProc.Signal(os.Interrupt)
		})

		It("returns a value metric containing uptime after specified time", func() {
			Eventually(fakeEventEmitter.GetMessages).Should(HaveLen(1))

			metric := fakeEventEmitter.GetMessages()[0].Event.(*events.ValueMetric)
			Expect(metric.Name).To(Equal(proto.String("uptime")))
			Expect(metric.Unit).To(Equal(proto.String("seconds")))
		})

		It("reports increasing uptime value", func() {
			Eventually(fakeEventEmitter.GetMessages).Should(HaveLen(1))
			metric := fakeEventEmitter.GetMessages()[0].Event.(*events.ValueMetric)
			uptime := *(metric.Value)

			Eventually(getLatestUptime, "2s").Should(BeNumerically(">", uptime))
		})
	})

	It("stops the monitor and respective ticker", func() {
		Eventually(func() int { return len(fakeEventEmitter.GetMessages()) }).Should(BeNumerically(">=", 1))

		uptimeProc.Signal(os.Interrupt)

		current := getLatestUptime()
		Consistently(getLatestUptime, 2).Should(Equal(current))
	})
})

func getLatestUptime() float64 {
	lastMsgIndex := len(fakeEventEmitter.GetMessages()) - 1
	return *fakeEventEmitter.GetMessages()[lastMsgIndex].Event.(*events.ValueMetric).Value
}
