package planner_test

import (
	"errors"
	"lib/models"
	"lib/rules"
	"vxlan-policy-agent/fakes"
	"vxlan-policy-agent/planner"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/gardenfakes"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Planner", func() {
	var (
		policyPlanner  *planner.VxlanPolicyPlanner
		gardenClient   *gardenfakes.FakeClient
		policyClient   *fakes.PolicyClient
		fakeContainer1 *gardenfakes.FakeContainer
		fakeContainer2 *gardenfakes.FakeContainer
		logger         *lagertest.TestLogger
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeContainer1 = &gardenfakes.FakeContainer{}
		fakeContainer2 = &gardenfakes.FakeContainer{}
		gardenClient = &gardenfakes.FakeClient{}
		policyClient = &fakes.PolicyClient{}

		fakeContainer1.InfoReturns(garden.ContainerInfo{
			ContainerIP: "10.255.1.2",
			Properties:  map[string]string{"network.app_id": "some-app-guid"},
		}, nil)
		fakeContainer2.InfoReturns(garden.ContainerInfo{
			ContainerIP: "10.255.1.3",
			Properties:  map[string]string{"network.app_id": "some-other-app-guid"},
		}, nil)
		gardenClient.ContainersReturns([]garden.Container{
			fakeContainer1,
			fakeContainer2,
		}, nil)

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

		policyPlanner = &planner.VxlanPolicyPlanner{
			Logger:       logger,
			GardenClient: gardenClient,
			PolicyClient: policyClient,
			VNI:          42,
		}
	})
	Describe("GetRules", func() {
		It("gets every container's properties from the garden client", func() {
			_, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(gardenClient.ContainersCallCount()).To(Equal(1))
			Expect(gardenClient.ContainersArgsForCall(0)).To(Equal(garden.Properties{}))
		})
		It("gets policies from the policy server", func() {
			_, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
		})
		It("returns local and remote rules based on the policies", func() {
			ruleset, err := policyPlanner.GetRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(ruleset).To(ConsistOf([]rules.GenericRule{
				// remote allows
				{
					Properties: []string{
						"-i", "flannel.42",
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
						"-i", "flannel.42",
						"-d", "10.255.1.3",
						"-p", "tcp",
						"--dport", "1234",
						"-m", "mark", "--mark", "0xAA",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:some-app-guid dst:some-other-app-guid",
					},
				},
				// local allows
				{
					Properties: []string{
						"-i", "cni-flannel0",
						"--source", "10.255.1.2",
						"-d", "10.255.1.3",
						"-p", "tcp",
						"--dport", "1234",
						"--jump", "ACCEPT",
						"-m", "comment", "--comment", "src:some-app-guid dst:some-other-app-guid",
					},
				},
				// tagging
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
		Context("when getting containers from garden fails", func() {
			BeforeEach(func() {
				gardenClient.ContainersReturns(nil, errors.New("banana"))
			})
			It("logs and returns the error", func() {
				_, err := policyPlanner.GetRules()

				Expect(err).To(MatchError("banana"))
				Expect(logger).To(gbytes.Say("garden-client-containers.*banana"))
			})
		})
		Context("when getting container info fails", func() {
			BeforeEach(func() {
				fakeContainer1.InfoReturns(garden.ContainerInfo{}, errors.New("potato"))
			})
			It("logs and returns the error", func() {
				_, err := policyPlanner.GetRules()

				Expect(err).To(MatchError("potato"))
				Expect(logger).To(gbytes.Say("container-info.*potato"))
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
