package planner

import (
	"lib/models"
	"lib/rules"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPolicies() ([]models.Policy, error)
}

type VxlanPolicyPlanner struct {
	Logger       lager.Logger
	GardenClient garden.Client
	PolicyClient policyClient
	VNI          int
}

type Container struct {
	Handle  string
	IP      string
	GroupID string
}

func getContainersMap(allContainers []garden.Container) (map[string][]string, error) {
	containers := map[string][]string{}

	for _, container := range allContainers {
		info, err := container.Info()
		if err != nil {
			return nil, err
		}
		properties := info.Properties
		groupID := properties["network.policy_group_id"]

		containers[groupID] = append(containers[groupID], info.ContainerIP)
	}

	return containers, nil
}

func (p *VxlanPolicyPlanner) GetRules() ([]rules.Rule, error) {
	properties := garden.Properties{}
	gardenContainers, err := p.GardenClient.Containers(properties)
	if err != nil {
		p.Logger.Error("garden-client-containers", err)
		return nil, err
	}

	containers, err := getContainersMap(gardenContainers)
	if err != nil {
		p.Logger.Error("container-info", err)
		return nil, err
	}
	p.Logger.Debug("got-containers", lager.Data{"containers": containers})

	policies, err := p.PolicyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("policy-client-get-policies", err)
		return nil, err
	}

	marksRuleset := []rules.Rule{}
	filterRuleset := []rules.Rule{}

	for _, policy := range policies {
		srcContainerIPs, srcOk := containers[policy.Source.ID]
		dstContainerIPs, dstOk := containers[policy.Destination.ID]

		if dstOk {
			// there are some containers on this host that are dests for the policy
			for _, dstContainerIP := range dstContainerIPs {
				filterRuleset = append(
					filterRuleset,
					rules.NewMarkAllowRule(
						dstContainerIP,
						policy.Destination.Protocol,
						policy.Destination.Port,
						policy.Source.Tag,
						policy.Source.ID,
						policy.Destination.ID,
					),
				)
			}
		}

		if srcOk {
			// there are some containers on this host that are sources for the policy
			for _, srcContainerIP := range srcContainerIPs {
				marksRuleset = append(
					marksRuleset,
					rules.NewMarkSetRule(srcContainerIP, policy.Source.Tag, policy.Source.ID),
				)
			}
		}
	}
	ruleset := append(marksRuleset, filterRuleset...)
	p.Logger.Debug("generated-rules", lager.Data{"rules": ruleset})
	return ruleset, nil
}
