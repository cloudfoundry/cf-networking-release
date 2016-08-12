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
		groupID := properties["network.app_id"]

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
	p.Logger.Info("got-containers", lager.Data{"containers": containers})

	policies, err := p.PolicyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("policy-client-get-policies", err)
		return nil, err
	}

	ruleset := []rules.Rule{}

	for _, policy := range policies {
		srcContainerIPs, srcOk := containers[policy.Source.ID]
		dstContainerIPs, dstOk := containers[policy.Destination.ID]

		if dstOk {
			for _, dstContainerIP := range dstContainerIPs {
				ruleset = append(
					ruleset,
					rules.NewRemoteAllowRule(
						p.VNI,
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
			for _, srcContainerIP := range srcContainerIPs {
				ruleset = append(
					ruleset,
					rules.NewGBPTagRule(srcContainerIP, policy.Source.Tag, policy.Source.ID),
				)
			}
		}

		if srcOk && dstOk {
			for _, srcContainerIP := range srcContainerIPs {
				for _, dstContainerIP := range dstContainerIPs {
					ruleset = append(
						ruleset,
						rules.NewLocalAllowRule(
							srcContainerIP,
							dstContainerIP,
							policy.Destination.Protocol,
							policy.Destination.Port,
							policy.Source.ID,
							policy.Destination.ID,
						),
					)
				}
			}
		}
	}
	p.Logger.Info("generated-rules", lager.Data{"rules": ruleset})
	return ruleset, nil
}
