package planner_test

import (
	"errors"
	"lib/datastore"
	libfakes "lib/fakes"
	"lib/models"
	"lib/rules"
	"vxlan-policy-agent/enforcer"
	"vxlan-policy-agent/fakes"
	"vxlan-policy-agent/planner"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Planner", func() {
	var (
		policyPlanner        *planner.VxlanPolicyPlanner
		policyClient         *fakes.PolicyClient
		policyServerResponse []models.Policy
		store                *libfakes.Datastore
		timeMetricsEmitter   *fakes.TimeMetricsEmitter
		logger               *lagertest.TestLogger
		chain                enforcer.Chain
		data                 map[string]datastore.Container
		loggingStateGetter   *fakes.LoggingStateGetter
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		policyClient = &fakes.PolicyClient{}
		timeMetricsEmitter = &fakes.TimeMetricsEmitter{}
		loggingStateGetter = &fakes.LoggingStateGetter{}

		store = &libfakes.Datastore{}

		data = make(map[string]datastore.Container)
		data["container-id-1"] = datastore.Container{
			Handle: "container-id-1",
			IP:     "10.255.1.2",
			Metadata: map[string]interface{}{
				"policy_group_id": "some-app-guid",
			},
		}
		data["container-id-2"] = datastore.Container{
			Handle: "container-id-2",
			IP:     "10.255.1.3",
			Metadata: map[string]interface{}{
				"policy_group_id": "some-other-app-guid",
			},
		}
		data["container-id-3"] = datastore.Container{
			Handle: "container-id-3",
			IP:     "10.255.1.4",
		}

		store.ReadAllReturns(data, nil)

		policyServerResponse = []models.Policy{
			{
				Source: models.Source{
					ID:  "some-app-guid",
					Tag: "AA",
				},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Port:     1234,
					Protocol: "tcp",
				},
			},
			{
				Source: models.Source{
					ID:  "another-app-guid",
					Tag: "BB",
				},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Port:     5555,
					Protocol: "udp",
				},
			},
			{
				Source: models.Source{
					ID:  "some-other-app-guid",
					Tag: "CC",
				},
				Destination: models.Destination{
					ID:       "yet-another-app-guid",
					Port:     6534,
					Protocol: "udp",
				},
			},
		}
		policyClient.GetPoliciesReturns(policyServerResponse, nil)

		chain = enforcer.Chain{
			Table:       "some-table",
			ParentChain: "INPUT",
			Prefix:      "some-prefix",
		}

		policyPlanner = &planner.VxlanPolicyPlanner{
			Logger:            logger,
			Datastore:         store,
			PolicyClient:      policyClient,
			VNI:               42,
			CollectionEmitter: timeMetricsEmitter,
			Chain:             chain,
			LoggingState:      loggingStateGetter,
		}
	})

	Describe("GetRulesAndChain", func() {
		It("gets every container's properties from the datastore", func() {
			_, err := policyPlanner.GetRulesAndChain()
			Expect(err).NotTo(HaveOccurred())

			Expect(store.ReadAllCallCount()).To(Equal(1))
		})

		It("gets policies from the policy server", func() {
			_, err := policyPlanner.GetRulesAndChain()
			Expect(err).NotTo(HaveOccurred())

			Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
		})

		Context("when iptables logging is disabled", func() {
			BeforeEach(func() {
				loggingStateGetter.IsEnabledReturns(false)
			})
			It("returns all the rules but no logging rules", func() {
				rulesWithChain, err := policyPlanner.GetRulesAndChain()
				Expect(err).NotTo(HaveOccurred())
				Expect(rulesWithChain.Chain).To(Equal(chain))

				Expect(rulesWithChain.Rules).To(ConsistOf([]rules.IPTablesRule{
					// allow based on mark
					{
						"-d", "10.255.1.3",
						"-p", "udp",
						"--dport", "5555",
						"-m", "mark", "--mark", "0xBB",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:another-app-guid_dst:some-other-app-guid",
					},
					{
						"-d", "10.255.1.3",
						"-p", "tcp",
						"--dport", "1234",
						"-m", "mark", "--mark", "0xAA",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:some-app-guid_dst:some-other-app-guid",
					},
					// set tags on all outgoing packets, regardless of local vs remote
					{
						"--source", "10.255.1.2",
						"--jump", "MARK", "--set-xmark", "0xAA",
						"-m", "comment", "--comment", "src:some-app-guid",
					},
					{
						"--source", "10.255.1.3",
						"--jump", "MARK", "--set-xmark", "0xCC",
						"-m", "comment", "--comment", "src:some-other-app-guid",
					},
				}))
			})
		})

		Context("when iptables logging is enabled", func() {
			BeforeEach(func() {
				loggingStateGetter.IsEnabledReturns(true)
			})
			It("returns all the rules including logging rules", func() {
				rulesWithChain, err := policyPlanner.GetRulesAndChain()
				Expect(err).NotTo(HaveOccurred())
				Expect(rulesWithChain.Chain).To(Equal(chain))

				Expect(rulesWithChain.Rules).To(gomegamatchers.ContainSequence([]rules.IPTablesRule{
					// LOG bb allow based on mark
					{
						"-d", "10.255.1.3",
						"-p", "udp",
						"--dport", "5555",
						"-m", "mark", "--mark", "0xBB",
						"-m", "limit", "--limit", "2/min",
						"--jump", "LOG", "--log-prefix", `"OK_BB_some-other-app-guid"`,
					},
					// allow bb based on mark
					{
						"-d", "10.255.1.3",
						"-p", "udp",
						"--dport", "5555",
						"-m", "mark", "--mark", "0xBB",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:another-app-guid_dst:some-other-app-guid",
					},
				}))
				Expect(rulesWithChain.Rules).To(gomegamatchers.ContainSequence([]rules.IPTablesRule{
					// LOG aa allow based on mark
					{
						"-d", "10.255.1.3",
						"-p", "tcp",
						"--dport", "1234",
						"-m", "mark", "--mark", "0xAA",
						"-m", "limit", "--limit", "2/min",
						"--jump", "LOG", "--log-prefix", `"OK_AA_some-other-app-guid"`,
					},
					// allow aa based on mark
					{
						"-d", "10.255.1.3",
						"-p", "tcp",
						"--dport", "1234",
						"-m", "mark", "--mark", "0xAA",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:some-app-guid_dst:some-other-app-guid",
					},
				}))
				Expect(rulesWithChain.Rules).To(ContainElement(rules.IPTablesRule{
					// set tags on all outgoing packets, regardless of local vs remote
					"--source", "10.255.1.2",
					"--jump", "MARK", "--set-xmark", "0xAA",
					"-m", "comment", "--comment", "src:some-app-guid",
				}))
				Expect(rulesWithChain.Rules).To(ContainElement(rules.IPTablesRule{
					"--source", "10.255.1.3",
					"--jump", "MARK", "--set-xmark", "0xCC",
					"-m", "comment", "--comment", "src:some-other-app-guid",
				}))
			})
		})

		It("returns all mark set rules before any mark filter rules", func() {
			rulesWithChain, err := policyPlanner.GetRulesAndChain()
			Expect(err).NotTo(HaveOccurred())
			Expect(rulesWithChain.Rules).To(HaveLen(4))
			Expect(rulesWithChain.Rules[0]).To(ContainElement("--set-xmark"))
			Expect(rulesWithChain.Rules[1]).To(ContainElement("--set-xmark"))
			Expect(rulesWithChain.Rules[2]).To(ContainElement("ACCEPT"))
			Expect(rulesWithChain.Rules[3]).To(ContainElement("ACCEPT"))
		})

		It("emits time metrics", func() {
			_, err := policyPlanner.GetRulesAndChain()
			Expect(err).NotTo(HaveOccurred())
			Expect(timeMetricsEmitter.EmitAllCallCount()).To(Equal(1))
		})

		Context("when the policies are returned from the server in a different order", func() {
			var reversed []models.Policy
			BeforeEach(func() {
				for i, _ := range policyServerResponse {
					reversed = append(reversed, policyServerResponse[len(policyServerResponse)-i-1])
				}
			})

			It("the order of the rules is not affected", func() {
				rulesWithChain, err := policyPlanner.GetRulesAndChain()
				Expect(err).NotTo(HaveOccurred())
				policyClient.GetPoliciesReturns(reversed, nil)
				rulesWithChain2, err := policyPlanner.GetRulesAndChain()
				Expect(err).NotTo(HaveOccurred())

				Expect(rulesWithChain).To(Equal(rulesWithChain2))
			})
		})

		Context("when a container's metadata is missing required key policy group id", func() {
			BeforeEach(func() {
				data["container-id-fruit"] = datastore.Container{
					Handle: "container-id-fruit",
					IP:     "10.255.1.5",
					Metadata: map[string]interface{}{
						"fruit": "banana",
					},
				}
			})

			It("logs an error for that container and returns rules for other containers", func() {
				rulesWithChain, err := policyPlanner.GetRulesAndChain()
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say("container-metadata-policy-group-id.*container-id-fruit.*Container.*metadata.*policy_group_id.*CloudController.*restage"))

				Expect(rulesWithChain.Chain).To(Equal(chain))
				Expect(rulesWithChain.Rules).To(HaveLen(4))
			})
		})

		Context("when getting containers from datastore fails", func() {
			BeforeEach(func() {
				store.ReadAllReturns(nil, errors.New("banana"))
			})

			It("logs and returns the error", func() {
				_, err := policyPlanner.GetRulesAndChain()
				Expect(err).To(MatchError("banana"))
				Expect(logger).To(gbytes.Say("datastore.*banana"))
			})
		})

		Context("when getting policies fails", func() {
			BeforeEach(func() {
				policyClient.GetPoliciesReturns(nil, errors.New("kiwi"))
			})

			It("logs and returns the error", func() {
				_, err := policyPlanner.GetRulesAndChain()
				Expect(err).To(MatchError("kiwi"))
				Expect(logger).To(gbytes.Say("policy-client-get-policies.*kiwi"))
			})
		})
	})
})
