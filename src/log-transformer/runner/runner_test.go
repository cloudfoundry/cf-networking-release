package runner_test

import (
	"errors"
	"log-transformer/merger"
	"log-transformer/parser"
	"log-transformer/runner"
	"log-transformer/runner/fakes"
	"os"
	"strconv"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	"github.com/hpcloud/tail"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Runner", func() {
	var (
		lines          chan *tail.Line
		fakeParser     *fakes.KernelLogParser
		fakeMerger     *fakes.LogMerger
		logger         *lagertest.TestLogger
		iptablesLogger *lagertest.TestLogger
		logRunner      *runner.Runner
		logRunnerProc  ifrit.Process
	)

	BeforeEach(func() {
		lines = make(chan *tail.Line)
		fakeParser = &fakes.KernelLogParser{}
		fakeMerger = &fakes.LogMerger{}
		logger = lagertest.NewTestLogger("test")
		iptablesLogger = lagertest.NewTestLogger("iptables-test")

		logRunner = &runner.Runner{
			Lines:          lines,
			Parser:         fakeParser,
			Merger:         fakeMerger,
			Logger:         logger,
			IPTablesLogger: iptablesLogger,
		}
	})

	AfterEach(func() {
		logRunnerProc.Signal(os.Interrupt)
		Eventually(logRunnerProc.Wait()).Should(Receive())
		Expect(logger.Logs()[len(logger.Logs())-1]).To(LogsWith(lager.INFO, "test.exited"))
	})

	It("logs that it started and can be run in an ifrit process", func() {
		logRunnerProc = ifrit.Invoke(logRunner)
		Eventually(logRunnerProc.Ready()).Should(BeClosed())
		Eventually(func() int {
			return len(logger.Logs())
		}).Should(BeNumerically(">", 0))
		Expect(logger.Logs()[0]).To(LogsWith(lager.INFO, "test.started"))
	})

	Context("when the kernel log gets an iptables message", func() {
		BeforeEach(func() {
			fakeParser.IsIPTablesLogDataReturns(true)

			fakeParser.ParseReturns(parser.ParsedData{
				SourceIP:      "source-ip",
				DestinationIP: "dest-ip",
			})

			fakeMerger.MergeReturns(merger.IPTablesLogData{
				Message: "some-message",
				Data:    lager.Data{"foo": "bar"},
			}, nil)
		})

		It("parses log line from the kernel log", func() {
			logRunnerProc = ifrit.Invoke(logRunner)
			go func() {
				lines <- &tail.Line{
					Text: "some-line",
				}
			}()

			Eventually(fakeParser.IsIPTablesLogDataCallCount).Should(Equal(1))
			Expect(fakeParser.IsIPTablesLogDataArgsForCall(0)).To(Equal("some-line"))

			Expect(fakeParser.ParseCallCount()).To(Equal(1))
			Expect(fakeParser.ParseArgsForCall(0)).Should(Equal("some-line"))

			Expect(fakeMerger.MergeCallCount()).To(Equal(1))
			Expect(fakeMerger.MergeArgsForCall(0)).To(Equal(parser.ParsedData{
				SourceIP:      "source-ip",
				DestinationIP: "dest-ip",
			}))

			Eventually(iptablesLogger.Logs).Should(HaveLen(1))
			Expect(iptablesLogger.Logs()[0]).To(SatisfyAll(
				LogsWith(lager.INFO, "iptables-test.some-message"),
				HaveLogData(SatisfyAll(
					HaveLen(1),
					HaveKeyWithValue("foo", "bar"),
				)),
			))
		})
	})

	Context("when the kernel log gets a non-iptables message", func() {
		BeforeEach(func() {
			fakeParser.IsIPTablesLogDataReturns(false)
		})
		It("parses log line from the kernel log", func() {
			logRunnerProc = ifrit.Invoke(logRunner)
			go func() {
				lines <- &tail.Line{
					Text: "some-line",
				}
			}()

			Eventually(fakeParser.IsIPTablesLogDataCallCount).Should(Equal(1))
			Expect(fakeParser.IsIPTablesLogDataArgsForCall(0)).To(Equal("some-line"))

			Expect(fakeParser.ParseCallCount()).To(Equal(0))
			Expect(fakeMerger.MergeCallCount()).To(Equal(0))

			Expect(iptablesLogger.Logs()).To(HaveLen(0))
		})
	})

	Context("when the kernel log gets multiple messages", func() {
		BeforeEach(func() {
			fakeParser.IsIPTablesLogDataStub = func(line string) bool {
				i, err := strconv.Atoi(line)
				Expect(err).NotTo(HaveOccurred())
				return i%2 == 0
			}
		})
		It("logs all the iptables log messages", func() {
			logRunnerProc = ifrit.Invoke(logRunner)
			go func() {
				for i := 0; i < 8; i++ {
					lines <- &tail.Line{
						Text: strconv.Itoa(i),
					}
				}
			}()

			Eventually(fakeParser.IsIPTablesLogDataCallCount).Should(Equal(8))

			Eventually(fakeParser.ParseCallCount).Should(Equal(4))
			Expect(fakeParser.ParseArgsForCall(0)).Should(Equal("0"))
			Expect(fakeParser.ParseArgsForCall(1)).Should(Equal("2"))
			Expect(fakeParser.ParseArgsForCall(2)).Should(Equal("4"))
			Expect(fakeParser.ParseArgsForCall(3)).Should(Equal("6"))

			Eventually(fakeMerger.MergeCallCount).Should(Equal(4))

			Eventually(iptablesLogger.Logs).Should(HaveLen(4))
		})
	})

	Context("when the tailer returns a line with an error", func() {
		It("logs the error to the process's logger", func() {
			logRunnerProc = ifrit.Invoke(logRunner)
			go func() {
				lines <- &tail.Line{
					Text: "some-line",
					Err:  errors.New("banana"),
				}
			}()

			Eventually(logger.Logs).Should(HaveLen(2))
			Expect(logger.Logs()[1]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.tail-kernel-logs"),
				HaveLogData(SatisfyAll(
					HaveLen(1),
					HaveKeyWithValue("error", "banana"),
				)),
			))

			Expect(fakeParser.IsIPTablesLogDataCallCount()).To(Equal(0))
			Expect(fakeParser.ParseCallCount()).To(Equal(0))
			Expect(fakeMerger.MergeCallCount()).To(Equal(0))
			Expect(iptablesLogger.Logs()).To(HaveLen(0))
		})
	})

	// TODO when reading the store fails with an error
	// TODO test where we save ReadAll result and only re-read if we can't find src app when src on parser.Parse is present (or likewise can't find dst app when dst on parser.Parse is present)

	Context("when merging the logs fails", func() {
		BeforeEach(func() {
			fakeParser.IsIPTablesLogDataReturns(true)

			fakeParser.ParseReturns(parser.ParsedData{})

			fakeMerger.MergeReturns(merger.IPTablesLogData{}, errors.New("banana"))
		})
		It("logs the error to the process's logger", func() {
			logRunnerProc = ifrit.Invoke(logRunner)
			go func() {
				lines <- &tail.Line{
					Text: "some-line",
				}
			}()

			Eventually(logger.Logs).Should(HaveLen(2))
			Expect(logger.Logs()[1]).To(SatisfyAll(
				LogsWith(lager.ERROR, "test.merge-kernel-logs"),
				HaveLogData(SatisfyAll(
					HaveLen(1),
					HaveKeyWithValue("error", "banana"),
				)),
			))

			Expect(iptablesLogger.Logs()).To(HaveLen(0))
		})
	})
})

var LogsWith = func(level lager.LogLevel, msg string) types.GomegaMatcher {
	return And(
		WithTransform(func(log lager.LogFormat) string {
			return log.Message
		}, Equal(msg)),
		WithTransform(func(log lager.LogFormat) lager.LogLevel {
			return log.LogLevel
		}, Equal(level)),
	)
}

var HaveLogData = func(nextMatcher types.GomegaMatcher) types.GomegaMatcher {
	return WithTransform(func(log lager.LogFormat) lager.Data {
		return log.Data
	}, nextMatcher)
}
