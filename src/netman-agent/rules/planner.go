package rules

import (
	"fmt"
	"netman-agent/models"
	"strconv"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/store_reader.go --fake-name StoreReader . storeReader
type storeReader interface {
	GetContainers() map[string][]models.Container
}

//go:generate counterfeiter -o ../fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPolicies() ([]models.Policy, error)
}

type Planner struct {
	Logger       lager.Logger
	storeReader  storeReader
	policyClient policyClient
	VNI          int
	LocalSubnet  string
	RuleEnforcer RuleEnforcer
}

//go:generate counterfeiter -o ../fakes/rule_enforcer.go --fake-name RuleEnforcer . RuleEnforcer
type RuleEnforcer interface {
	Enforce(chain string, r []Rule) error
}

func New(logger lager.Logger, storeReader storeReader, policyClient policyClient, vni int, localSubnet string, ruleEnforcer RuleEnforcer) *Planner {
	return &Planner{
		Logger:       logger,
		storeReader:  storeReader,
		policyClient: policyClient,
		VNI:          vni,
		LocalSubnet:  localSubnet,
		RuleEnforcer: ruleEnforcer,
	}
}

func (u *Planner) DefaultLocalRules() []Rule {
	r := []Rule{}

	r = append(r, GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"-j", "ACCEPT",
		},
	}, GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-s", u.LocalSubnet,
			"-d", u.LocalSubnet,
			"-m", "limit", "--limit", "2/min",
			"-j", "LOG",
			"--log-prefix", "DROP_LOCAL",
		},
	}, GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-s", u.LocalSubnet,
			"-d", u.LocalSubnet,
			"-j", "DROP",
		},
	})

	return r
}

func (u *Planner) DefaultRemoteRules() []Rule {
	r := []Rule{}

	r = append(r, GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", u.VNI),
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"-j", "ACCEPT",
		},
	}, GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", u.VNI),
			"-m", "limit", "--limit", "2/min",
			"-j", "LOG",
			"--log-prefix", "DROP_REMOTE",
		},
	}, GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", u.VNI),
			"-j", "DROP",
		},
	})

	return r
}

func (p *Planner) Update() error {
	rules, err := p.Rules()
	if err != nil {
		return err
	}
	err = p.RuleEnforcer.Enforce("netman--forward-", rules)
	if err != nil {
		return err
	}
	return nil
}

func (p *Planner) Rules() ([]Rule, error) {
	containers := p.storeReader.GetContainers()
	policies, err := p.policyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("get-policies", err)
		return nil, fmt.Errorf("get policies failed: %s", err)
	}

	rules := []Rule{}

	for _, policy := range policies {
		srcContainers, srcOk := containers[policy.Source.ID]
		dstContainers, dstOk := containers[policy.Destination.ID]

		// local dest
		if dstOk {
			for _, dstContainer := range dstContainers {
				rules = append(rules, GenericRule{
					Properties: []string{
						"-i", fmt.Sprintf("flannel.%d", p.VNI),
						"-d", dstContainer.IP,
						"-p", policy.Destination.Protocol,
						"--dport", strconv.Itoa(policy.Destination.Port),
						"-m", "mark", "--mark", fmt.Sprintf("0x%s", policy.Source.Tag),
						"-j", "ACCEPT",
					},
				})
			}
		}

		if srcOk {
			for _, srcContainer := range srcContainers {
				rules = append(rules, GenericRule{
					Properties: []string{
						"-s", srcContainer.IP,
						"-j", "MARK", "--set-xmark", fmt.Sprintf("0x%s", policy.Source.Tag),
					},
				})
			}
		}

		// local
		if srcOk && dstOk {
			for _, srcContainer := range srcContainers {
				for _, dstContainer := range dstContainers {
					rules = append(rules, GenericRule{
						Properties: []string{
							"-i", "cni-flannel0",
							"-s", srcContainer.IP,
							"-d", dstContainer.IP,
							"-p", policy.Destination.Protocol,
							"--dport", strconv.Itoa(policy.Destination.Port),
							"-j", "ACCEPT",
						},
					})
				}
			}
		}
	}

	return rules, nil
}
