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
	"github.com/nats-io/go-nats/bench"
	toputils "github.com/nats-io/nats-top/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
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

	It(fmt.Sprintf("NATS subscriptions when publishing %d messages", config.NumMessages), Serial, func() {
		exp := gmeasure.NewExperiment("Sending and receiving messages")
		AddReportEntry(exp.Name, exp)

		i := 0
		exp.Sample(func(idx int) {
			i++
			fmt.Printf("Iteration %d\n", i)

			By("building a benchmark of subscribers listening on service-discovery.register")
			benchMarkNatsSubMap := collectNatsSubscriberConnectionInfo(SdcRegisterTopic)

			By("publish messages onto service-discovery.register")
			natsBenchmark := runNatsBenchmarker(NatsRun{0, SdcRegisterTopic})
			generateBenchmarkGinkgoReport(exp, natsBenchmark, "")
			Expect(int(natsBenchmark.MsgCnt)).To(Equal(config.NumMessages))

			By("building an updated benchmark of subscribers listening on service-discovery.register")
			natsSubMap := collectNatsSubscriberConnectionInfo(SdcRegisterTopic)

			By("making sure service-discovery.register subscribers received every published message", func() {
				for key, benchMarkVal := range benchMarkNatsSubMap {
					Expect(int(natsSubMap[key].OutMsgs-benchMarkVal.OutMsgs)).To(Equal(config.NumMessages), fmt.Sprintf("Benchmark: %+v \\n \\n Got %+v", benchMarkVal, natsSubMap[key]))
				}
			})
		}, gmeasure.SamplingConfig{N: 3})
	})

	It("SDC subscription CPU load does not greatly increase Nats CPU", Serial, func() {
		var medianJustExternalRoutes float64
		var meanJustExternalRoutes float64

		var medianExternalAndInternalRoutes float64
		var meanExternalAndInternalRoutes float64

		exp := gmeasure.NewExperiment("")
		AddReportEntry(exp.Name, exp)

		i := 0
		exp.Sample(func(idx int) {
			i++
			fmt.Printf("Iteration %d\n", i)

			medianJustExternalRoutes, meanJustExternalRoutes = runCpuAndMemProfle(exp, "External Routes", NatsRun{2, "router.register"})
			medianExternalAndInternalRoutes, meanExternalAndInternalRoutes = runCpuAndMemProfle(exp, "Both External and Internal Routes", NatsRun{2, "router.register"}, NatsRun{0, "service-discovery.register"})

			Expect(medianExternalAndInternalRoutes).Should(BeNumerically("~", medianJustExternalRoutes, 30.00))
			Expect(meanExternalAndInternalRoutes).Should(BeNumerically("~", meanJustExternalRoutes, 30.00))

		}, gmeasure.SamplingConfig{N: 3})
	})
})

func runCpuAndMemProfle(exp *gmeasure.Experiment, prefix string, natsRuns ...NatsRun) (median, mean float64) {
	By(fmt.Sprintf("Running benchmark for %s", prefix), func() {
		cpuChannel := make(chan float64)
		stopCpuProfiling := make(chan struct{})

		go startProfiler(exp, addPrefix(prefix, "CPU"), addPrefix(prefix, "Memory"), stopCpuProfiling, cpuChannel)

		var cpuValuesSlice []float64
		cpuMapperDone := make(chan struct{})

		go func() {
			for cpu := range cpuChannel {
				cpuValuesSlice = append(cpuValuesSlice, cpu)
			}
			close(cpuMapperDone)
		}()

		natsBenchmark := runNatsBenchmarker(natsRuns...)
		close(stopCpuProfiling)

		generateBenchmarkGinkgoReport(exp, natsBenchmark, prefix)

		<-cpuMapperDone
		var err error
		median, err = stats.Median(cpuValuesSlice)
		Expect(err).NotTo(HaveOccurred())
		mean, err = stats.Mean(cpuValuesSlice)
		Expect(err).NotTo(HaveOccurred())

		exp.RecordValue(addPrefix(prefix, "median"), median)
		exp.RecordValue(addPrefix(prefix, "mean"), mean)
	})

	closeSubscribers()

	return median, mean
}

func closeSubscribers() {
	for _, natsConn := range subscriberNatsConnections {
		if natsConn == nil {
			continue
		}

		natsConn.Close()
	}

	subscriberNatsConnections = []*nats.Conn{}
}

func startProfiler(exp *gmeasure.Experiment, cpuKey, memKey string, stopCpuProfiling chan struct{}, cpuValuesChannel chan float64) {
	defer GinkgoRecover()
	natsTopEngine := toputils.NewEngine(config.NatsURL, config.NatsMonitoringPort, 1000, 0)
	natsTopEngine.SetupHTTP()

	go func() {
		defer GinkgoRecover()
		Expect(natsTopEngine.MonitorStats()).To(Succeed())
	}()

	for {
		select {
		case natsStats, ok := <-natsTopEngine.StatsCh:
			if !ok {
				continue
			}
			exp.RecordValue(cpuKey, natsStats.Varz.CPU)
			exp.RecordValue(memKey, float64(natsStats.Varz.Mem))
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

func generateBenchmarkGinkgoReport(exp *gmeasure.Experiment, bm *bench.Benchmark, prefix string) {
	if bm.Pubs.HasSamples() {
		if len(bm.Pubs.Samples) > 1 {
			exp.RecordValue(addPrefix(prefix, "PubStats"), float64(bm.Pubs.Rate()), gmeasure.Units("msgs/sec"))
			for i, stat := range bm.Pubs.Samples {
				exp.RecordValue(addPrefix(prefix, fmt.Sprintf("Pub %d", i)), float64(stat.MsgCnt), gmeasure.Annotation(fmt.Sprintf("publisher # %d", i)))
			}
			exp.RecordValue(addPrefix(prefix, "min"), float64(bm.Pubs.MinRate()))
			exp.RecordValue(addPrefix(prefix, "avg"), float64(bm.Pubs.AvgRate()))
			exp.RecordValue(addPrefix(prefix, "max"), float64(bm.Pubs.MaxRate()))
			exp.RecordValue(addPrefix(prefix, "stddev"), float64(bm.Pubs.StdDev()))
		}
	}
}

func addPrefix(prefix, metricName string) string {
	if prefix != "" {
		return fmt.Sprintf("%s %s", prefix, metricName)
	}

	return metricName
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
