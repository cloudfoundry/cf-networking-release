package performance_test

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-top/util"
	"github.com/nats-io/nats/bench"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const SdcRegisterTopic = "service-discovery.register"

const ProfilerExternalRoutesCPUKey = "cpu_external_routes"
const ProfilerExternalAndInternalRoutesCPUKey = "cpu_external_and_internal_routes"
const ProfilerExternalRoutesMemKey = "mem_external_routes"
const ProfilerExternalAndInternalRoutesMemKey = "mem_external_and_internal_routes"

type NatsRun struct {
	subscriberCount int
	topic           string
}

var (
	subLock                   sync.Mutex
	subscriberNatsConnections []*nats.Conn
)

var _ = Describe("NatsPerformance", func() {
	BeforeEach(func() {
		subscriberNatsConnections = []*nats.Conn{}
	})

	AfterEach(func() {
		closeSubscribers()
	})

	Measure(fmt.Sprintf("NATS subscriptions when publishing %d messages", config.NumMessages), func(b Benchmarker) {
		By("building a benchmark of subscribers listening on service-discovery.register")
		benchMarkNatsSubMap := collectNatsSubscriberConnectionInfo(SdcRegisterTopic)

		By("publish messages onto service-discovery.register")
		natsBenchmark := runNatsBenchmarker(NatsRun{0, SdcRegisterTopic})
		generateBenchmarkGinkgoReport(b, natsBenchmark)
		Expect(int(natsBenchmark.MsgCnt)).To(Equal(config.NumMessages))

		By("building an updated benchmark of subscribers listening on service-discovery.register")
		natsSubMap := collectNatsSubscriberConnectionInfo(SdcRegisterTopic)

		By("making sure service-discovery.register subscribers received every published message", func() {
			for key, benchMarkVal := range benchMarkNatsSubMap {
				Expect(int(natsSubMap[key].OutMsgs-benchMarkVal.OutMsgs)).To(Equal(config.NumMessages), fmt.Sprintf("Benchmark: %+v \\n \\n Got %+v", benchMarkVal, natsSubMap[key]))
			}
		})

	}, 1)

	Measure("SDC subscription CPU load does not greatly increase Nats CPU", func(b Benchmarker) {
		var medianJustExternalRoutes float64
		var meanJustExternalRoutes float64

		var medianExternalAndInternalRoutes float64
		var meanExternalAndInternalRoutes float64

		By("Running benchmark for just external routes", func() {
			cpuChannel := make(chan float64)
			stopCpuProfiling := make(chan struct{})

			go startProfiler(b, ProfilerExternalRoutesCPUKey, ProfilerExternalRoutesMemKey, stopCpuProfiling, cpuChannel)

			var cpuValuesJustExternalRoutesSlice []float64
			cpuMapperDone := make(chan struct{})

			go func() {
				for cpu := range cpuChannel {
					cpuValuesJustExternalRoutesSlice = append(cpuValuesJustExternalRoutesSlice, cpu)
				}
				close(cpuMapperDone)
			}()

			natsBenchmark := runNatsBenchmarker(NatsRun{2, "router.register"})
			close(stopCpuProfiling)

			generateBenchmarkGinkgoReport(b, natsBenchmark)

			<-cpuMapperDone
			var err error
			medianJustExternalRoutes, err = stats.Median(cpuValuesJustExternalRoutesSlice)
			Expect(err).NotTo(HaveOccurred())
			meanJustExternalRoutes, err = stats.Mean(cpuValuesJustExternalRoutesSlice)
			Expect(err).NotTo(HaveOccurred())

		})

		By("closing subscribers from previous nats benchmark run", func() {
			closeSubscribers()
		})

		By("Running benchmark for both external and internal routes", func() {
			cpuChannel := make(chan float64)
			stopCpuProfiling := make(chan struct{})

			go startProfiler(b, ProfilerExternalAndInternalRoutesCPUKey, ProfilerExternalAndInternalRoutesMemKey, stopCpuProfiling, cpuChannel)
			var cpuValueExternalAndInternalRoutesSlice []float64
			cpuMapperDone := make(chan struct{})

			go func() {
				for cpu := range cpuChannel {
					cpuValueExternalAndInternalRoutesSlice = append(cpuValueExternalAndInternalRoutesSlice, cpu)
				}

				close(cpuMapperDone)
			}()

			natsBenchmarkExternalAndInternalRoutes := runNatsBenchmarker(NatsRun{2, "router.register"}, NatsRun{0, "service-discovery.register"})
			close(stopCpuProfiling)

			generateBenchmarkGinkgoReport(b, natsBenchmarkExternalAndInternalRoutes)

			<-cpuMapperDone
			var err error
			medianExternalAndInternalRoutes, err = stats.Median(cpuValueExternalAndInternalRoutesSlice)
			Expect(err).NotTo(HaveOccurred())
			meanExternalAndInternalRoutes, err = stats.Mean(cpuValueExternalAndInternalRoutesSlice)
			Expect(err).NotTo(HaveOccurred())
		})

		b.RecordValue("medianExternalAndInternalRoutes", medianExternalAndInternalRoutes)
		b.RecordValue("meanExternalAndInternalRoutes", meanExternalAndInternalRoutes)
		b.RecordValue("medianJustExternalRoutes", medianJustExternalRoutes)
		b.RecordValue("meanJustExternalRoutes", meanJustExternalRoutes)

		Expect(medianExternalAndInternalRoutes).Should(BeNumerically("~", medianJustExternalRoutes, 15.00))
		Expect(meanExternalAndInternalRoutes).Should(BeNumerically("~", meanJustExternalRoutes, 15.00))
	}, 1)
})

func closeSubscribers() {
	for _, natsConn := range subscriberNatsConnections {
		if natsConn == nil {
			continue
		}

		natsConn.Close()
	}

	subscriberNatsConnections = []*nats.Conn{}
}

func startProfiler(ginkgoBenchmarker Benchmarker, cpuKey, memKey string, stopCpuProfiling chan struct{}, cpuValuesChannel chan float64) {
	defer GinkgoRecover()
	natsTopEngine := toputils.NewEngine(config.NatsURL, config.NatsMonitoringPort, 1000, 0)
	natsTopEngine.SetupHTTP()

	go func() {
		defer GinkgoRecover()
		Expect(natsTopEngine.MonitorStats()).ShouldNot(HaveOccurred())
	}()

	for {
		select {
		case natsStats, ok := <-natsTopEngine.StatsCh:
			if !ok {
				continue
			}
			ginkgoBenchmarker.RecordValue(cpuKey, natsStats.Varz.CPU)
			ginkgoBenchmarker.RecordValue(memKey, float64(natsStats.Varz.Mem))
			Expect(int(natsStats.Varz.SlowConsumers)).To(Equal(0))
			cpuValuesChannel <- natsStats.Varz.CPU
		case <-stopCpuProfiling:
			close(natsTopEngine.ShutdownCh)
			close(cpuValuesChannel)
			return
		}

	}
}

func runNatsBenchmarker(natsRuns ...NatsRun) *bench.Benchmark {
	opts := nats.GetDefaultOptions()
	opts.Servers = strings.Split(config.NatsURL, ",")
	opts.User = config.NatsUsername
	opts.Password = config.NatsPassword
	for i, s := range opts.Servers {
		opts.Servers[i] = "nats://" + strings.Trim(s, " ") + ":" + strconv.Itoa(config.NatsPort)
	}

	var startwg sync.WaitGroup

	totalSubscriberCount := 0
	for _, natsRun := range natsRuns {
		totalSubscriberCount += natsRun.subscriberCount
	}

	natsBenchmark := bench.NewBenchmark("SDC Nats Benchmark", totalSubscriberCount, config.NumPublisher*len(natsRuns))

	startSubscribers := time.Now()
	for _, natsRun := range natsRuns {
		startwg.Add(natsRun.subscriberCount)
		for i := 0; i < natsRun.subscriberCount; i++ {
			go runSubscriber(natsRun.topic, &startwg, opts)
		}

		startwg.Add(config.NumPublisher)
		pubCounts := bench.MsgsPerClient(config.NumMessages, config.NumPublisher)
		for _, pubCount := range pubCounts {
			go runPublisher(natsRun.topic, natsBenchmark, &startwg, opts, pubCount, NATS_MSG_SIZE)
		}

	}

	startwg.Wait()
	for _, natsConn := range subscriberNatsConnections {
		natsConn.Close()
		end := time.Now()
		time.Sleep(1 * time.Second)
		natsBenchmark.AddSubSample(bench.NewSample(config.NumMessages, NATS_MSG_SIZE, startSubscribers, end, natsConn))
	}

	natsBenchmark.Close()
	return natsBenchmark
}

func generateBenchmarkGinkgoReport(b Benchmarker, bm *bench.Benchmark) {
	if bm.Pubs.HasSamples() {
		if len(bm.Pubs.Samples) > 1 {
			b.RecordValue("PubStats", float64(bm.Pubs.Rate()), "msgs/sec")
			for i, stat := range bm.Pubs.Samples {
				b.RecordValue(fmt.Sprintf("Pub %d", i), float64(stat.MsgCnt), fmt.Sprintf("publisher # %d", i))
			}
			b.RecordValue("min", float64(bm.Pubs.MinRate()))
			b.RecordValue("avg", float64(bm.Pubs.AvgRate()))
			b.RecordValue("max", float64(bm.Pubs.MaxRate()))
			b.RecordValue("stddev", float64(bm.Pubs.StdDev()))
		}
	}

}

func collectNatsSubscriberConnectionInfo(subscriber string) map[uint64]server.ConnInfo {
	serviceDiscoverySubs := map[uint64]server.ConnInfo{}

	timeoutChan := time.After(10 * time.Second)
	natsTopEngine := toputils.NewEngine(config.NatsURL, config.NatsMonitoringPort, 1000, 1)
	natsTopEngine.DisplaySubs = true
	natsTopEngine.SetupHTTP()

	go func() {
		defer GinkgoRecover()
		Expect(natsTopEngine.MonitorStats()).To(Succeed())
	}()

	for {
		select {
		case stats, ok := <-natsTopEngine.StatsCh:
			if !ok {
				continue
			}
			for _, statsConn := range stats.Connz.Conns {
				if strings.Contains(strings.Join(statsConn.Subs, ","), subscriber) {
					serviceDiscoverySubs[statsConn.Cid] = server.ConnInfo(statsConn)
				}
			}
		case <-timeoutChan:
			close(natsTopEngine.ShutdownCh)
			return serviceDiscoverySubs
		}
	}
}

func runPublisher(subject string, benchmark *bench.Benchmark, startwg *sync.WaitGroup, opts nats.Options, numMsgs int, msgSize int) {
	defer GinkgoRecover()
	defer startwg.Done()
	nc, err := opts.Connect()
	Expect(err).NotTo(HaveOccurred())

	var msg []byte
	if msgSize > 0 {
		msg = make([]byte, msgSize)
	}

	start := time.Now()

	for i := 0; i < numMsgs; i++ {
		nc.Publish(subject, msg)
	}
	nc.Flush()
	nc.Close()
	end := time.Now()
	time.Sleep(1 * time.Second)
	benchmark.AddPubSample(bench.NewSample(numMsgs, msgSize, start, end, nc))
}

func runSubscriber(subject string, startwg *sync.WaitGroup, opts nats.Options) {
	defer GinkgoRecover()
	defer startwg.Done()
	nc, err := opts.Connect()
	Expect(err).NotTo(HaveOccurred())

	_, err = nc.Subscribe(subject, func(*nats.Msg) {})
	Expect(err).NotTo(HaveOccurred())

	subLock.Lock()
	defer subLock.Unlock()
	subscriberNatsConnections = append(subscriberNatsConnections, nc)
}
