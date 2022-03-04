package asg_syncer_test

import (
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/asg_syncer"
	"code.cloudfoundry.org/policy-server/asg_syncer/fakes"
	"code.cloudfoundry.org/policy-server/cc_client"
	ccfakes "code.cloudfoundry.org/policy-server/cc_client/fakes"
	"code.cloudfoundry.org/policy-server/store"
	dbfakes "code.cloudfoundry.org/policy-server/store/fakes"
	uaafakes "code.cloudfoundry.org/policy-server/uaa_client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ASGSyncer", func() {
	var (
		fakeUAAClient     *uaafakes.UAAClient
		fakeCCClient      *ccfakes.CCClient
		logger            *lagertest.TestLogger
		fakeStore         *dbfakes.SecurityGroupsStore
		pollInterval      time.Duration
		fakeMetricsSender *fakes.MetricsSender
	)
	BeforeEach(func() {
		fakeStore = &dbfakes.SecurityGroupsStore{}
		fakeUAAClient = &uaafakes.UAAClient{}
		fakeCCClient = &ccfakes.CCClient{}
		logger = lagertest.NewTestLogger("test")
		pollInterval = time.Millisecond
		fakeMetricsSender = &fakes.MetricsSender{}
	})
	Describe("NewASGSyncer()", func() {
		asgSyncer := asg_syncer.NewASGSyncer(logger, fakeStore, fakeUAAClient, fakeCCClient, pollInterval, fakeMetricsSender, time.Millisecond)

		Expect(asgSyncer).To(Equal(&asg_syncer.ASGSyncer{
			Logger:        logger,
			Store:         fakeStore,
			UAAClient:     fakeUAAClient,
			CCClient:      fakeCCClient,
			MetricsSender: fakeMetricsSender,
			RetryDeadline: time.Millisecond,
		}))
	})
	Describe("asgSyncer.Poll()", func() {
		var asgSyncer *asg_syncer.ASGSyncer
		BeforeEach(func() {
			asgSyncer = asg_syncer.NewASGSyncer(logger, fakeStore, fakeUAAClient, fakeCCClient, pollInterval, fakeMetricsSender, time.Millisecond)
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
		Describe("asgSyncer.Run()", func() {
			var (
				signals chan os.Signal
				ready   chan struct{}

				retChan chan error
			)

			BeforeEach(func() {
				signals = make(chan os.Signal)
				ready = make(chan struct{})

				retChan = make(chan error)
			})

			It("polls at poll interval", func() {
				go func() {
					retChan <- asgSyncer.Run(signals, ready)
				}()

				Eventually(ready).Should(BeClosed())
				Eventually(fakeUAAClient.GetTokenCallCount()).Should(BeNumerically(">", 1))

				Consistently(retChan).ShouldNot(Receive())

				signals <- os.Interrupt
				Eventually(retChan).Should(Receive(nil))
			})

			Context("when the poller func errors", func() {
				BeforeEach(func() {
					fakeUAAClient.GetTokenReturns("", fmt.Errorf("banana"))
				})

				It("logs the error and returns", func() {
					err := asgSyncer.Run(signals, ready)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Errorf("banana")))
					Expect(logger).To(gbytes.Say("asg-sync-cycle.*banana"))
				})
			})
		})

		It("polls properly", func() {
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
		Context("metrics", func() {
			It("emits ASG metrics", func() {
				err := asgSyncer.Poll()
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(2))
				metricName, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(metricName).To(Equal("SecurityGroupsRetrievalFromCCTime"))
				metricName, _ = fakeMetricsSender.SendDurationArgsForCall(1)
				Expect(metricName).To(Equal("SecurityGroupsTotalSyncTime"))
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
				Context("if changes are detected during capi pagination", func() {
					BeforeEach(func() {
						fakeCCClient.GetSecurityGroupsReturns([]cc_client.SecurityGroupResource{}, error(cc_client.NewUnstableSecurityGroupListError(fmt.Errorf("unstable list"))))
					})
					It("doesn't return an error", func() {
						err := asgSyncer.Poll()
						Expect(err).ToNot(HaveOccurred())
					})
					It("doesn't update the database", func() {
						asgSyncer.Poll()
						Expect(fakeStore.ReplaceCallCount()).To(Equal(0))
					})
					Context("if changes keep happening past the retry deadline", func() {
						It("throws an error", func() {
							Eventually(func(g Gomega) {
								err := asgSyncer.Poll()
								g.Expect(err).To(MatchError(fmt.Errorf("unable to retrieve a consistent listing of security groups from CAPI after '1ms': unstable list")))
							}).Should(Succeed())
							Expect(fakeCCClient.GetSecurityGroupsCallCount()).To(BeNumerically(">", 1))
						})
					})
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
