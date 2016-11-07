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
)

var _ = Describe("Planner", func() {
	var (
		policyPlanner      *planner.VxlanPolicyPlanner
		policyClient       *fakes.PolicyClient
		store              *libfakes.Datastore
		timeMetricsEmitter *fakes.TimeMetricsEmitter
		logger             *lagertest.TestLogger
		chain              enforcer.Chain
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		policyClient = &fakes.PolicyClient{}
		timeMetricsEmitter = &fakes.TimeMetricsEmitter{}

		store = &libfakes.Datastore{}

		data := make(map[string]datastore.Container)
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

		policyClient.GetPoliciesReturns([]models.Policy{
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
		}, nil)

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
		}
	})
	Describe("GetRules", func() {
		It("gets every container's properties from the datastore", func() {
			_, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(store.ReadAllCallCount()).To(Equal(1))
		})
		It("gets policies from the policy server", func() {
			_, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
		})
		It("returns all the rules", func() {
			rulesWithChain, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())
			Expect(rulesWithChain.Chain).To(Equal(chain))

			Expect(rulesWithChain.Rules).To(ConsistOf([]rules.GenericRule{
				// allow based on mark
				{
					Properties: []string{
						"-d", "10.255.1.3",
						"-p", "udp",
						"--dport", "5555",
						"-m", "mark", "--mark", "0xBB",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:another-app-guid dst:some-other-app-guid",
					},
				},
				{
					Properties: []string{
						"-d", "10.255.1.3",
						"-p", "tcp",
						"--dport", "1234",
						"-m", "mark", "--mark", "0xAA",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:some-app-guid dst:some-other-app-guid",
					},
				},
				// set tags on all outgoing packets, regardless of local vs remote
				{
					Properties: []string{
						"--source", "10.255.1.2",
						"--jump", "MARK", "--set-xmark", "0xAA",
						"-m", "comment", "--comment", "src:some-app-guid",
					},
				},
				{
					Properties: []string{
						"--source", "10.255.1.3",
						"--jump", "MARK", "--set-xmark", "0xCC",
						"-m", "comment", "--comment", "src:some-other-app-guid",
					},
				},
			}))
		})
		It("returns all mark set rules before any mark filter rules", func() {
			rulesWithChain, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())
			Expect(rulesWithChain.Rules).To(HaveLen(4))
			Expect(rulesWithChain.Rules[0].(rules.GenericRule).Properties).To(ContainElement("--set-xmark"))
			Expect(rulesWithChain.Rules[1].(rules.GenericRule).Properties).To(ContainElement("--set-xmark"))
			Expect(rulesWithChain.Rules[2].(rules.GenericRule).Properties).To(ContainElement("ACCEPT"))
			Expect(rulesWithChain.Rules[3].(rules.GenericRule).Properties).To(ContainElement("ACCEPT"))
		})
		It("emits time metrics", func() {
			_, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(timeMetricsEmitter.EmitAllCallCount()).To(Equal(1))
		})
		Context("when getting containers from datastore fails", func() {
			BeforeEach(func() {
				store.ReadAllReturns(nil, errors.New("banana"))
			})
			It("logs and returns the error", func() {
				_, err := policyPlanner.GetRules()

				Expect(err).To(MatchError("banana"))
				Expect(logger).To(gbytes.Say("datastore.*banana"))
			})
		})
		Context("when getting policies fails", func() {
			BeforeEach(func() {
				policyClient.GetPoliciesReturns(nil, errors.New("kiwi"))
			})
			It("logs and returns the error", func() {
				_, err := policyPlanner.GetRules()

				Expect(err).To(MatchError("kiwi"))
				Expect(logger).To(gbytes.Say("policy-client-get-policies.*kiwi"))
			})
		})
	})
})
