package planner_test

import (
	"errors"
	"netman-agent/fakes"
	"netman-agent/models"
	"netman-agent/planner"
	"netman-agent/rules"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Planner", func() {
	var (
		p            *planner.Planner
		storeReader  *fakes.StoreReader
		policyClient *fakes.PolicyClient
		enforcer     *fakes.RuleEnforcer
		logger       *lagertest.TestLogger
	)

	BeforeEach(func() {
		storeReader = &fakes.StoreReader{}
		policyClient = &fakes.PolicyClient{}
		enforcer = &fakes.RuleEnforcer{}
		logger = lagertest.NewTestLogger("test")

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

		p = planner.New(
			logger,
			storeReader,
			policyClient,
			42,
			"8.8.8.0/24",
			"8.8.0.0/16",
			enforcer,
		)
	})

	Describe("DefaultLocalRules", func() {
		It("creates a list of default local rules and enforces them", func() {
			err := p.DefaultLocalRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(enforcer.EnforceCallCount()).To(Equal(1))
			tableArg, parentChain, chainName, localRules := enforcer.EnforceArgsForCall(0)

			Expect(tableArg).To(Equal("filter"))
			Expect(parentChain).To(Equal("FORWARD"))
			Expect(chainName).To(Equal("netman--local-"))

			Expect(len(localRules)).To(Equal(3))
			Expect(localRules).To(ConsistOf([]rules.GenericRule{
				{Properties: []string{
					"-i", "cni-flannel0",
					"-m", "state", "--state", "ESTABLISHED,RELATED",
					"-j", "ACCEPT",
				}},
				{Properties: []string{
					"-i", "cni-flannel0",
					"-s", "8.8.8.0/24",
					"-d", "8.8.8.0/24",
					"-m", "limit", "--limit", "2/min",
					"-j", "LOG",
					"--log-prefix", "DROP_LOCAL: ",
				}},
				{Properties: []string{
					"-i", "cni-flannel0",
					"-s", "8.8.8.0/24",
					"-d", "8.8.8.0/24",
					"-j", "DROP",
				}},
			}))
		})

		It("returns when there is an error", func() {
			expectedError := errors.New("banana")
			enforcer.EnforceReturns(expectedError)
			err := p.DefaultLocalRules()
			Expect(err).To(Equal(expectedError))
		})
	})

	Describe("DefaultRemoteRules", func() {
		It("creates a list of default remote rules and enforces them", func() {
			err := p.DefaultRemoteRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(enforcer.EnforceCallCount()).To(Equal(1))
			tableArg, parentChain, chainName, remoteRules := enforcer.EnforceArgsForCall(0)

			Expect(tableArg).To(Equal("filter"))
			Expect(parentChain).To(Equal("FORWARD"))
			Expect(chainName).To(Equal("netman--remote-"))

			Expect(len(remoteRules)).To(Equal(3))
			Expect(remoteRules).To(ConsistOf([]rules.GenericRule{
				{Properties: []string{
					"-i", "flannel.42",
					"-m", "state", "--state", "ESTABLISHED,RELATED",
					"-j", "ACCEPT",
				}},
				{Properties: []string{
					"-i", "flannel.42",
					"-m", "limit", "--limit", "2/min",
					"-j", "LOG",
					"--log-prefix", "DROP_REMOTE: ",
				}},
				{Properties: []string{
					"-i", "flannel.42",
					"-j", "DROP",
				}},
			}))
		})

		It("returns when there is an error", func() {
			expectedError := errors.New("banana")
			enforcer.EnforceReturns(expectedError)
			err := p.DefaultRemoteRules()
			Expect(err).To(Equal(expectedError))
		})
	})

	Describe("DefaultEgressRules", func() {
		It("creates the default rules to allow connectivity to the internet", func() {
			err := p.DefaultEgressRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(enforcer.EnforceCallCount()).To(Equal(1))
			tableArg, parentChain, chainName, egressRules := enforcer.EnforceArgsForCall(0)

			Expect(tableArg).To(Equal("nat"))
			Expect(parentChain).To(Equal("POSTROUTING"))
			Expect(chainName).To(Equal("netman--postrout-"))

			Expect(len(egressRules)).To(Equal(1))
			Expect(egressRules).To(ConsistOf([]rules.GenericRule{
				{Properties: []string{
					"-s", "8.8.8.0/24",
					"!", "-d", "8.8.0.0/16",
					"-j", "MASQUERADE",
				}},
			}))
		})
	})

	Describe("Rules", func() {
		It("gets the policies and containers", func() {
			_, err := p.Rules()
			Expect(err).NotTo(HaveOccurred())

			Expect(storeReader.GetContainersCallCount()).To(Equal(1))
			Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
		})

		It("converts policies into rule structs", func() {
			r, err := p.Rules()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(r)).To(Equal(4))
			Expect(r).To(ConsistOf([]rules.GenericRule{{
				Properties: []string{
					"-i", "flannel.42",
					"-d", "8.8.8.9",
					"-p", "tcp",
					"--dport", "5555",
					"-m", "mark", "--mark", "0x0123",
					"-j", "ACCEPT",
					"-m", "comment", "--comment", "src:some-app-guid dst:some-other-app-guid",
				},
			}, {
				Properties: []string{
					"-s", "8.8.8.8",
					"-j", "MARK", "--set-xmark", "0x0123",
					"-m", "comment", "--comment", "src:some-app-guid",
				},
			}, {
				Properties: []string{
					"-i", "cni-flannel0",
					"-s", "8.8.8.8",
					"-d", "8.8.8.9",
					"-p", "tcp",
					"--dport", "5555",
					"-j", "ACCEPT",
					"-m", "comment", "--comment", "src:some-app-guid dst:some-other-app-guid",
				},
			}, {
				Properties: []string{
					"-i", "flannel.42",
					"-d", "8.8.8.9",
					"-p", "tcp",
					"--dport", "5555",
					"-m", "mark", "--mark", "0x0124",
					"-j", "ACCEPT",
					"-m", "comment", "--comment", "src:some-remote-app dst:some-other-app-guid",
				},
			}}))
		})

		Context("when the policy client fails", func() {
			BeforeEach(func() {
				policyClient.GetPoliciesReturns(nil, errors.New("banana"))
			})

			It("returns and logs the error", func() {
				_, err := p.Rules()
				Expect(err).To(MatchError("get policies failed: banana"))
				Expect(logger).To(gbytes.Say(`get-policies.*banana`))
			})
		})
	})
})
