package asg_syncer_test

import (
	"fmt"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/locket"
	locketconfig "code.cloudfoundry.org/locket/cmd/locket/config"
	locketrunner "code.cloudfoundry.org/locket/cmd/locket/testrunner"
	"code.cloudfoundry.org/locket/lock"
	locketmodels "code.cloudfoundry.org/locket/models"
	"code.cloudfoundry.org/policy-server/asg_syncer"
	"code.cloudfoundry.org/policy-server/cc_client"
	ccfakes "code.cloudfoundry.org/policy-server/cc_client/fakes"
	"code.cloudfoundry.org/policy-server/store"
	dbfakes "code.cloudfoundry.org/policy-server/store/fakes"
	uaafakes "code.cloudfoundry.org/policy-server/uaa_client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("ASGSyncer", func() {
	var (
		fakeUAAClient *uaafakes.UAAClient
		fakeCCClient  *ccfakes.CCClient
		logger        *lagertest.TestLogger
		fakeStore     *dbfakes.SecurityGroupsStore
	)
	BeforeEach(func() {
		fakeStore = &dbfakes.SecurityGroupsStore{}
		fakeUAAClient = &uaafakes.UAAClient{}
		fakeCCClient = &ccfakes.CCClient{}
		logger = lagertest.NewTestLogger("test")
	})
	Describe("NewASGSyncer()", func() {
		asgSyncer := asg_syncer.NewASGSyncer(logger, fakeStore, fakeUAAClient, fakeCCClient)

		Expect(asgSyncer).To(Equal(&asg_syncer.ASGSyncer{
			Logger:    logger,
			Store:     fakeStore,
			UAAClient: fakeUAAClient,
			CCClient:  fakeCCClient,
		}))
	})
	Describe("asgSyncer.Sync()", func() {
		var asgSyncer *asg_syncer.ASGSyncer
		BeforeEach(func() {
			asgSyncer = asg_syncer.NewASGSyncer(logger, fakeStore, fakeUAAClient, fakeCCClient)
			fakeUAAClient.GetTokenReturns("fake-token", nil)
			fakeCCClient.GetSecurityGroupsReturns([]cc_client.SecurityGroupResource{{
				GUID:            "first-guid",
				Name:            "asg-1",
				GloballyEnabled: cc_client.SecurityGroupGloballyEnabled{Running: true, Staging: true},
				Rules: []cc_client.SecurityGroupRule{{
					Protocol:    "ICMP",
					Destination: "10.10.10.10/32",
					Code:        4,
					Type:        1,
					Description: "fake icmp rule",
					Log:         false,
				}, {
					Protocol:    "TCP",
					Destination: "20.20.20.20/32",
					Ports:       "80-1024",
					Description: "fake tcp rule",
					Log:         true,
				}},
				Relationships: cc_client.SecurityGroupRelationships{},
			}, {
				GUID: "second-guid",
				Name: "asg-2",
				Rules: []cc_client.SecurityGroupRule{{
					Protocol:    "UDP",
					Destination: "0.0.0/0",
					Ports:       "53",
					Description: "fake dns rule",
					Log:         true,
				}},
				Relationships: cc_client.SecurityGroupRelationships{
					RunningSpaces: cc_client.SecurityGroupSpaceRelationship{
						Data: []map[string]string{{
							"guid": "space-1-guid",
						}, {
							"guid": "space-2-guid",
						}},
					},
					StagingSpaces: cc_client.SecurityGroupSpaceRelationship{
						Data: []map[string]string{{
							"guid": "space-3-guid",
						}},
					},
				},
			}}, nil)
		})
		It("pulls properly", func() {
			err := asgSyncer.Poll()
			Expect(err).To(BeNil())

			By("Getting a UAA token", func() {
				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			})
			By("Requesting data from CAPI", func() {
				Expect(fakeCCClient.GetSecurityGroupsCallCount()).To(Equal(1))
				Expect(fakeCCClient.GetSecurityGroupsArgsForCall(0)).To(Equal("fake-token"))
			})
			By("calling Replace() on the store", func() {
				Expect(fakeStore.ReplaceCallCount()).To(Equal(1))
			})
			By("Translating cc_client security group resources into store security groups", func() {
				Expect(fakeStore.ReplaceArgsForCall(0)).To(Equal([]store.SecurityGroup{{
					Guid:              "first-guid",
					Name:              "asg-1",
					StagingDefault:    true,
					RunningDefault:    true,
					Rules:             `[{"protocol":"ICMP","destination":"10.10.10.10/32","ports":"","type":1,"code":4,"description":"fake icmp rule","log":false},{"protocol":"TCP","destination":"20.20.20.20/32","ports":"80-1024","type":0,"code":0,"description":"fake tcp rule","log":true}]`,
					RunningSpaceGuids: []string{},
					StagingSpaceGuids: []string{},
				}, {
					Guid:              "second-guid",
					Name:              "asg-2",
					Rules:             `[{"protocol":"UDP","destination":"0.0.0/0","ports":"53","type":0,"code":0,"description":"fake dns rule","log":true}]`,
					RunningSpaceGuids: []string{"space-1-guid", "space-2-guid"},
					StagingSpaceGuids: []string{"space-3-guid"},
				}}))
			})
		})
		Context("when acquiring a lock", func() {
			Context("and the lock is already taken", func() {
				var locketProcess ifrit.Process
				var competingLockProcess ifrit.Process
				var cr *consulrunner.ClusterRunner
				var locketAddress string

				BeforeEach(func() {

					locketBinPath, err := gexec.Build("code.cloudfoundry.org/locket/cmd/locket", "-race")
					Expect(err).NotTo(HaveOccurred())

					locketPort := 123456
					locketAddress = fmt.Sprintf("localhost:%d", locketPort)

					cr = consulrunner.NewClusterRunner(
						consulrunner.ClusterRunnerConfig{
							StartingPort: int(123455),
							NumNodes:     1,
							Scheme:       "http",
						},
					)

					locketRunner := locketrunner.NewLocketRunner(locketBinPath, func(cfg *locketconfig.LocketConfig) {
						cfg.ConsulCluster = cr.ConsulCluster()
						cfg.DatabaseConnectionString = "user:password@/locket"
						cfg.DatabaseDriver = "mysql"
						cfg.ListenAddress = locketAddress
					})
					locketProcess = ginkgomon.Invoke(locketRunner)

					competingIdentifier := &locketmodels.Resource{
						Key:      "policy-server-asg-syncher",
						Owner:    "some-server",
						TypeCode: locketmodels.LOCK,
						Type:     locketmodels.LockType,
					}

					competingClient, _ := locket.NewClient(logger,
						locketrunner.ClientLocketConfig(),
					)
					var competingLock = lock.NewLockRunner(
						logger,
						competingClient,
						competingIdentifier,
						locket.DefaultSessionTTLInSeconds,
						clock.NewClock(),
						locket.SQLRetryInterval,
					)
					competingLockProcess = ginkgomon.Invoke(competingLock)
				})
				AfterEach(func() {
					ginkgomon.Kill(locketProcess)
					ginkgomon.Kill(competingLockProcess)
				})

				FIt("doesn't poll capi or update the database", func() {
					err := asgSyncer.Poll()
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeCCClient.GetSecurityGroupsCallCount()).To(Equal(0))
					Expect(fakeStore.ReplaceCallCount()).To(Equal(0))
				})
				It("debug logs that another policy-server has the lock", func() {
					Expect(logger.Logs()).To(Equal("something"))
				})
			})
			Context("and the lock is obtained successfully", func() {
				It("polls CAPI and updates the database", func() {
					err := asgSyncer.Poll()
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeCCClient.GetSecurityGroupsCallCount()).To(Equal(1))
					Expect(fakeStore.ReplaceCallCount()).To(Equal(1))
				})
				It("info logs that it is the leader now", func() {
					Expect(logger.Logs()).To(Equal("something"))
				})
			})
			Context("and an error occurs obtaining the lock", func() {
				It("returns a relevant error", func() {
					err := asgSyncer.Poll()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("uaa error"))

				})
				It("doesn't poll capi or update the database", func() {
					asgSyncer.Poll()
					Expect(fakeCCClient.GetSecurityGroupsCallCount()).To(Equal(0))
					Expect(fakeStore.ReplaceCallCount()).To(Equal(0))
				})
			})

		})

		Context("when errors occur", func() {

			Context("getting a UAA token", func() {
				BeforeEach(func() {
					fakeUAAClient.GetTokenReturns("", fmt.Errorf("uaa error"))
				})
				It("returns a relevant error", func() {
					err := asgSyncer.Poll()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("uaa error"))

				})
				It("doesn't poll capi or update the database", func() {
					asgSyncer.Poll()
					Expect(fakeCCClient.GetSecurityGroupsCallCount()).To(Equal(0))
					Expect(fakeStore.ReplaceCallCount()).To(Equal(0))
				})
			})

			Context("polling CAPI", func() {
				BeforeEach(func() {
					fakeCCClient.GetSecurityGroupsReturns([]cc_client.SecurityGroupResource{}, fmt.Errorf("capi error"))
				})
				It("returns a relevant error", func() {
					err := asgSyncer.Poll()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("capi error"))
				})
				It("doesn't update the database", func() {
					asgSyncer.Poll()
					Expect(fakeStore.ReplaceCallCount()).To(Equal(0))
				})
			})

			Context("when CAPI returns bad relationship data", func() {
				BeforeEach(func() {
					fakeCCClient.GetSecurityGroupsReturns([]cc_client.SecurityGroupResource{{
						Name: "bad-asg",
						GUID: "bad-asg-guid",
						Relationships: cc_client.SecurityGroupRelationships{
							RunningSpaces: cc_client.SecurityGroupSpaceRelationship{
								Data: []map[string]string{{"blarg": "blargh"}},
							},
						},
					}}, nil)
				})
				It("returns a relevant error", func() {
					err := asgSyncer.Poll()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("no 'guid' found for running-space-relationship on asg 'bad-asg-guid'"))

				})
				It("doesn't update the database", func() {
					asgSyncer.Poll()
					Expect(fakeStore.ReplaceCallCount()).To(Equal(0))
				})
			})

			Context("replacing data in the store", func() {
				BeforeEach(func() {
					fakeStore.ReplaceReturns(fmt.Errorf("db error"))
				})
				It("returns a relevant error", func() {
					err := asgSyncer.Poll()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("db error"))
				})
			})
		})
	})
})
