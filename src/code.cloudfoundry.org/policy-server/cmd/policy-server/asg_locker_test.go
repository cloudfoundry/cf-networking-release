package main

import (
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/locket"
	locketconfig "code.cloudfoundry.org/locket/cmd/locket/config"
	locketrunner "code.cloudfoundry.org/locket/cmd/locket/testrunner"
	locketmodels "code.cloudfoundry.org/locket/models"
	fakesyncer "code.cloudfoundry.org/policy-server/asg_syncer/fakes"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("ASG Locker", func() {

	pollInterval := time.Millisecond

	var (
		fakeSyncer    *fakesyncer.ASGSyncer
		locketClient  locketmodels.LocketClient
		asgLocker     ifrit.Runner
		process       ifrit.Process
		locketProcess ifrit.Process
		logger        *lagertest.TestLogger
		dbConf        db.Config
		realDb        *db.ConnWrapper
		err           error
		locketAddress string
		locketPath    string
	)

	createLocketRunner := func() ifrit.Runner {
		dbConnectionString, err := dbConf.ConnectionString()
		Expect(err).NotTo(HaveOccurred())

		return locketrunner.NewLocketRunner(locketPath, func(cfg *locketconfig.LocketConfig) {
			cfg.DatabaseConnectionString = dbConnectionString
			cfg.DatabaseDriver = dbConf.Type
			cfg.ListenAddress = locketAddress
		})
	}

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("asg_locker_test_%d", time.Now().UnixNano())

		testhelpers.CreateDatabase(dbConf)

		realDb, err = db.NewConnectionPool(dbConf, 200, 0, 60*time.Minute, "ASG Locker Test", "ASG Locker Test", logger)
		Expect(err).NotTo(HaveOccurred())

		locketPort := ports.PickAPort()
		locketAddress = fmt.Sprintf("localhost:%d", locketPort)

		locketPath, err = gexec.Build("code.cloudfoundry.org/vendor/code.cloudfoundry.org/locket/cmd/locket", "-race")
		Expect(err).NotTo(HaveOccurred())

		locketProcess = ginkgomon.Invoke(createLocketRunner())

		locketConfig := locketrunner.ClientLocketConfig()
		locketConfig.LocketAddress = locketAddress
		locketClient, err = locket.NewClient(logger, locketConfig)
		Expect(err).NotTo(HaveOccurred())
		fakeSyncer = &fakesyncer.ASGSyncer{}
		fakeSyncer.PollCalls(func() error {
			logger.Info("POLLING")
			return nil
		})
		asgLocker = initASGLocker(logger, "test-uuid", pollInterval, pollInterval, 1, fakeSyncer, locketClient)
	})
	AfterEach(func() {
		ginkgomon.Kill(process, 2*time.Second)
		ginkgomon.Kill(locketProcess)
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testhelpers.RemoveDatabase(dbConf)
	})

	JustBeforeEach(func() {
		process = ifrit.Background(asgLocker)
	})

	Context("when a lock cannot be obtained", func() {
		Context("due to something else claming the lock", func() {
			var competingFakeSyncer *fakesyncer.ASGSyncer
			var competingLock ifrit.Runner
			var competingProcess ifrit.Process

			BeforeEach(func() {
				competingFakeSyncer = &fakesyncer.ASGSyncer{}
				competingLock = initASGLocker(logger, "competing-uuid", pollInterval, pollInterval, 1, competingFakeSyncer, locketClient)
				competingProcess = ifrit.Background(competingLock)
				// Ensure that competingProcess's ready channel is closed (meaning that it has obtained a lock, and
				// started the poller (both the asg-lock + asg-poller members of its process are ready)
				Eventually(competingProcess.Ready).Should(BeClosed())
				Eventually(competingFakeSyncer.PollCallCount).Should(BeNumerically(">", 0))
			})

			AfterEach(func() {
				ginkgomon.Kill(competingProcess)
			})

			It("does not acquire the lock", func() {
				// Ensure that competingProcess's ready channel is still open (meaning that it has not yet
				// obtained a lock. Since the asg-poller runner closes its ready channel immediately, and
				// is the last member of this ifrit group, we know that it has not started polling yet
				Consistently(process.Ready).ShouldNot(BeClosed())
				Eventually(logger.Logs).Should(ContainElement(ContainMessageFromOwner("test-uuid", "failed-to-acquire-lock")))
				Consistently(fakeSyncer.PollCallCount).Should(Equal(0))
			})

			Context("when the lock becomes available", func() {
				BeforeEach(func() {
					ginkgomon.Kill(competingProcess)
					// ensure the competingProcess has finished
					Eventually(competingProcess.Wait()).Should(Receive(nil))
				})

				It("acquires the lock", func() {
					// Ensure that process's ready channel is closed (meaning that it has finally obtained a lock, and
					// started the poller (both the asg-lock + asg-poller members of its process are ready)
					Eventually(process.Ready).Should(BeClosed())
					Eventually(logger.Logs).Should(ContainElement(ContainMessageFromOwner("test-uuid", "acquired-lock")))
					Eventually(fakeSyncer.PollCallCount).Should(BeNumerically(">", 0))
				})
			})
		})
	})
	Context("when the lock is lost", func() {
		var numSyncs int
		JustBeforeEach(func() {
			Eventually(process.Ready).Should(BeClosed())

			//Simulate losing a lock by killing the locket-server
			ginkgomon.Kill(locketProcess)
			locketProcess = nil
			numSyncs = fakeSyncer.PollCallCount()
		})
		It("restarts the poller + lock runners", func() {
			// ensure we haven't polled capi again
			Consistently(fakeSyncer.PollCallCount).Should(Equal(numSyncs))

			// re-run locket server and ensure it starts polling again
			locketProcess = ginkgomon.Invoke(createLocketRunner())
			Eventually(fakeSyncer.PollCallCount).Should(BeNumerically(">", numSyncs))
		})
		It("Logs that it lost lock + restarted the runner", func() {
			Eventually(logger.Logs).Should(ContainElement(ContainMessageFromOwner("test-uuid", "lost-lock")))
			Eventually(logger.LogMessages).Should(ContainElement(ContainSubstring("restarting-asg-locker")))
			Eventually(logger.Logs).Should(ContainElement(ContainMessageFromOwner("test-uuid", "failed-to-acquire-lock")))
		})
	})
})

type ContainMessageFromOwnerMatcher struct {
	Owner   string
	Message string
}

func (matcher *ContainMessageFromOwnerMatcher) Match(actual interface{}) (success bool, err error) {
	if logFormat, ok := actual.(lager.LogFormat); ok {
		if lockData, ok := logFormat.Data["lock"].(map[string]interface{}); ok {
			if lockData["owner"] == matcher.Owner && strings.Contains(logFormat.Message, matcher.Message) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (matcher *ContainMessageFromOwnerMatcher) FailureMessage(actual interface{}) (message string) {
	if logFormat, ok := actual.(lager.LogFormat); ok {
		if lockData, ok := logFormat.Data["lock"].(map[string]interface{}); ok {
			return fmt.Sprintf("Expected owner to match %s and message to match %s; got owner: %s, message: %s",
				matcher.Owner, matcher.Message, lockData["owner"], logFormat.Message,
			)
		}
	}
	return "Expected to get lager.LogFormat object"
}

func (matcher *ContainMessageFromOwnerMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected owner not to match %s and message not to match %s;",
		matcher.Owner, matcher.Message,
	)
}

func ContainMessageFromOwner(owner string, message string) types.GomegaMatcher {
	return &ContainMessageFromOwnerMatcher{
		Owner:   owner,
		Message: message,
	}
}
