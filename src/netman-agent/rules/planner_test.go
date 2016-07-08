package rules_test

import (
	"errors"
	"fmt"
	"netman-agent/fakes"
	"netman-agent/models"
	"netman-agent/rules"
	"strconv"
	"strings"
	"time"

	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Rules", func() {
	var (
		planner      *rules.Updater
		storeReader  *fakes.StoreReader
		policyClient *fakes.PolicyClient
		logger       *lagertest.TestLogger
		iptables     *fakes.IPTables
		table        string
		chain        string
		ruleSpec     []string
		pos          int
	)

	BeforeEach(func() {
		storeReader = &fakes.StoreReader{}
		policyClient = &fakes.PolicyClient{}
		logger = lagertest.NewTestLogger("test")
		iptables = &fakes.IPTables{}

		storeReader.GetContainersReturns(map[string][]models.Container{
			"some-app-guid": []models.Container{{
				ID: "some-container-id",
				IP: "8.8.8.8",
			}},
			"some-other-app-guid": []models.Container{{
				ID: "some-other-container-id",
				IP: "8.8.8.9",
			}},
		})

		policyClient.GetPoliciesReturns([]models.Policy{{
			models.Source{
				ID:  "some-app-guid",
				Tag: "0123",
			},
			models.Destination{
				ID:       "some-other-app-guid",
				Port:     5555,
				Protocol: "tcp",
			},
		}, {
			models.Source{
				ID:  "some-remote-app",
				Tag: "0124",
			},
			models.Destination{
				ID:       "some-other-app-guid",
				Port:     5555,
				Protocol: "tcp",
			},
		}}, nil)

		var err error
		planner, err = rules.New(
			logger,
			storeReader,
			policyClient,
			iptables,
			42,
			"8.8.8.0/24",
		)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("New", func() {
		It("creates the default chain with default rules and jumps to it", func() {
			Expect(iptables.NewChainCallCount()).To(Equal(1))
			Expect(iptables.AppendUniqueCallCount()).To(Equal(5))

			table, chain = iptables.NewChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netman--forward-default"))

			table, chain, ruleSpec = iptables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(ruleSpec).To(Equal([]string{"-j", "netman--forward-default"}))

			table, chain, ruleSpec = iptables.AppendUniqueArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netman--forward-default"))
			Expect(ruleSpec).To(Equal([]string{
				"-i", "cni-flannel0",
				"-m", "state", "--state", "ESTABLISHED,RELATED",
				"-j", "ACCEPT",
			}))

			table, chain, ruleSpec = iptables.AppendUniqueArgsForCall(2)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netman--forward-default"))
			Expect(ruleSpec).To(Equal([]string{
				"-i", "cni-flannel0",
				"-s", "8.8.8.0/24",
				"-d", "8.8.8.0/24",
				"-j", "DROP",
			}))
		})

		Context("when setting up new default chain fails", func() {
			BeforeEach(func() {
				iptables.AppendUniqueReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				_, err := rules.New(
					logger,
					storeReader,
					policyClient,
					iptables,
					42,
					"8.8.8.0/24",
				)
				Expect(err).To(MatchError("setting up default chain: banana"))
			})
		})
	})

	Describe("Rules", func() {
		It("gets the policies and containers", func() {
			_, err := planner.Rules()
			Expect(err).NotTo(HaveOccurred())

			Expect(storeReader.GetContainersCallCount()).To(Equal(1))
			Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
		})

		It("converts policies into rule structs", func() {
			r, err := planner.Rules()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(r)).To(Equal(4))
			Expect(r).To(ConsistOf(
				rules.RemoteAllowRule{SrcTag: "0123", DstIP: "8.8.8.9", Port: 5555, Proto: "tcp", VNI: 42, IPTables: iptables, Logger: logger},
				rules.LocalTagRule{SourceTag: "0123", SourceContainerIP: "8.8.8.8", IPTables: iptables, Logger: logger},
				rules.LocalAllowRule{SrcIP: "8.8.8.8", DstIP: "8.8.8.9", Port: 5555, Proto: "tcp", IPTables: iptables, Logger: logger},
				rules.RemoteAllowRule{SrcTag: "0124", DstIP: "8.8.8.9", Port: 5555, Proto: "tcp", VNI: 42, IPTables: iptables, Logger: logger},
			))
		})

		Context("when the policy client fails", func() {
			BeforeEach(func() {
				policyClient.GetPoliciesReturns(nil, errors.New("banana"))
			})

			It("returns and logs the error", func() {
				_, err := planner.Rules()
				Expect(err).To(MatchError("get policies failed: banana"))
				Expect(logger).To(gbytes.Say(`get-policies.*banana`))
			})
		})
	})

	Describe("Enforce", func() {
		var fakeRule *fakes.Rule

		BeforeEach(func() {
			fakeRule = &fakes.Rule{}
			fakeRule.ChainReturns("some-chain")
		})

		It("enforces all the rules it receives on the correct chain", func() {
			err := planner.Enforce([]rules.Rule{fakeRule})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRule.EnforceCallCount()).To(Equal(1))
			Expect(fakeRule.EnforceArgsForCall(0)).To(Equal("some-chain"))
		})

		Context("when there is an error enforcing a rule", func() {
			BeforeEach(func() {
				fakeRule.EnforceReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				err := planner.Enforce([]rules.Rule{fakeRule})
				Expect(err).To(MatchError("banana"))
			})
		})
	})

	Describe("Update", func() {
		It("updates the iptables forward chain rules for netman", func() {
			err := planner.Update()
			Expect(err).NotTo(HaveOccurred())

			Expect(iptables.NewChainCallCount()).To(Equal(2))
			table, myChain := iptables.NewChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(myChain).To(MatchRegexp("netman--forward-[0-9]{10}"))

			Expect(iptables.AppendUniqueCallCount()).To(Equal(9))

			table, chain, ruleSpec = iptables.AppendUniqueArgsForCall(7)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(MatchRegexp("netman--forward-[0-9]{10}"))
			Expect(ruleSpec).To(Equal([]string{
				"-i", "cni-flannel0",
				"-s", "8.8.8.8",
				"-d", "8.8.8.9",
				"-p", "tcp",
				"--dport", "5555",
				"-j", "ACCEPT",
			}))

			Expect(iptables.InsertCallCount()).To(Equal(1))
			table, chain, pos, ruleSpec = iptables.InsertArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(pos).To(Equal(1))
			Expect(ruleSpec).To(Equal([]string{"-j", myChain}))
		})

		Context("when there is already a forward chain from previous poll", func() {
			var oldChain string

			BeforeEach(func() {
				err := planner.Update()
				Expect(err).NotTo(HaveOccurred())

				table, oldChain = iptables.NewChainArgsForCall(1)

				time.Sleep(1 * time.Second)
				iptables.ListReturns([]string{
					"-P FORWARD ACCEPT",
					"-A FORWARD -i some-interface-j another-chain",
					fmt.Sprintf("-A FORWARD -i cni-flannel0 -j %s", oldChain),
					"-A FORWARD -i cni-flannel0 -j netman--forward-9999999999",
				}, nil)

				err = planner.Update()
				Expect(err).NotTo(HaveOccurred())
			})

			It("timestamps the chains", func() {
				Expect(oldChain).To(MatchRegexp("netman--forward-[0-9]{10}"))
				table, chain = iptables.NewChainArgsForCall(2)
				Expect(chain).To(MatchRegexp("netman--forward-[0-9]{10}"))

				oldTimestamp, err := strconv.Atoi(strings.TrimPrefix(oldChain, "netman--forward-"))
				Expect(err).NotTo(HaveOccurred())
				newTimestamp, err := strconv.Atoi(strings.TrimPrefix(chain, "netman--forward-"))
				Expect(err).NotTo(HaveOccurred())

				Expect(oldTimestamp).To(BeNumerically("<", newTimestamp))
			})

			It("deletes only the old chains", func() {
				Expect(iptables.DeleteCallCount()).To(Equal(1))
				table, chain, ruleSpec = iptables.DeleteArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("FORWARD"))
				Expect(ruleSpec).To(Equal([]string{"-j", oldChain}))

				Expect(iptables.ClearChainCallCount()).To(Equal(1))
				table, chain = iptables.ClearChainArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal(oldChain))

				Expect(iptables.DeleteChainCallCount()).To(Equal(1))
				table, chain = iptables.DeleteChainArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal(oldChain))
			})
		})

		It("logs the rules it is about to enforce", func() {
			err := planner.Update()
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say(`enforce-remote-rule.*{"dstIP":"8.8.8.9","port":5555,"proto":"tcp","srcTag":"0123","vni":42}`))
			Expect(logger).To(gbytes.Say(`set-local-tag.*{"srcIP":"8.8.8.8","srcTag":"0123"}`))
			Expect(logger).To(gbytes.Say(`enforce-local-rule.*{"dstIP":"8.8.8.9","port":5555,"proto":"tcp","srcIP":"8.8.8.8"}`))
			Expect(logger).To(gbytes.Say(`enforce-remote-rule.*{"dstIP":"8.8.8.9","port":5555,"proto":"tcp","srcTag":"0124","vni":42}`))
		})

		Context("when appending a new rule fails", func() {
			BeforeEach(func() {
				iptables.AppendUniqueReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				err := planner.Update()
				Expect(err).To(MatchError("appending rule: banana"))

				Expect(logger).To(gbytes.Say("append-rule.*banana"))
			})
		})

		Context("when there are errors cleaning up old rules", func() {
			BeforeEach(func() {
				iptables.ListReturns(nil, errors.New("blueberry"))
			})

			It("it logs and returns a useful error", func() {
				err := planner.Update()
				Expect(err).To(MatchError("listing forward rules: blueberry"))

				Expect(logger).To(gbytes.Say("cleanup-rules.*blueberry"))
			})
		})

		Context("when there are errors cleaning up old chains", func() {
			BeforeEach(func() {
				iptables.DeleteReturns(errors.New("banana"))
				iptables.ListReturns([]string{
					"-A FORWARD -i cni-flannel0 -j netman--forward-1111111111",
				}, nil)
			})

			It("returns a useful error", func() {
				err := planner.Update()
				Expect(err).To(MatchError("cleanup old chain: banana"))
			})
		})

		Context("when creating the new chain fails", func() {
			BeforeEach(func() {
				iptables.NewChainReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				err := planner.Update()
				Expect(err).To(MatchError("creating chain: banana"))

				Expect(logger).To(gbytes.Say("create-chain.*banana"))
			})
		})

		Context("when inserting the new chain fails", func() {
			BeforeEach(func() {
				iptables.InsertReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				err := planner.Update()
				Expect(err).To(MatchError("inserting chain: banana"))

				Expect(logger).To(gbytes.Say("insert-chain.*banana"))
			})
		})
	})
})
