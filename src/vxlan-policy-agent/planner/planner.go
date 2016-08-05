package planner

import (
	"fmt"
	"netman-agent/models"
	"netman-agent/policy_client"
	"netman-agent/rules"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

type VxlanPolicyPlanner struct {
	GardenClient   garden.Client
	PolicyClient   *policy_client.Client
	Logger         lager.Logger
	VNI            int
	LocalSubnet    string
	OverlayNetwork string
}

type Container struct {
	Handle  string
	IP      string
	GroupID string
}

func getContainersMap(allContainers []garden.Container) (map[string][]models.Container, error) {
	containers := map[string][]models.Container{}

	for _, container := range allContainers {
		info, err := container.Info()
		if err != nil {
			return nil, err
		}
		properties := info.Properties
		groupID := properties["network.app_id"]

		containers[groupID] = append(containers[groupID],
			models.Container{
				ID: container.Handle(),
				IP: info.ContainerIP,
			})
	}

	return containers, nil
}

func (p *VxlanPolicyPlanner) GetRules() ([]rules.Rule, error) {
	properties := garden.Properties{}
	gardenContainers, err := p.GardenClient.Containers(properties)
	if err != nil {
		return nil, err
	}

	containers, err := getContainersMap(gardenContainers)
	if err != nil {
		return nil, err
	}
	p.Logger.Info("got-containers", lager.Data{"containers": containers})

	policies, err := p.PolicyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("get-policies", err)
		return nil, fmt.Errorf("get policies failed: %s", err)
	}

	ruleset := []rules.Rule{}

	for _, policy := range policies {
		srcContainers, srcOk := containers[policy.Source.ID]
		dstContainers, dstOk := containers[policy.Destination.ID]

		if dstOk {
			for _, dstContainer := range dstContainers {
				ruleset = append(
					ruleset,
					rules.NewRemoteAllowRule(
						p.VNI,
						dstContainer.IP,
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
			for _, srcContainer := range srcContainers {
				ruleset = append(
					ruleset,
					rules.NewGBPTagRule(srcContainer.IP, policy.Source.Tag, policy.Source.ID),
				)
			}
		}

		if srcOk && dstOk {
			for _, srcContainer := range srcContainers {
				for _, dstContainer := range dstContainers {
					ruleset = append(
						ruleset,
						rules.NewLocalAllowRule(
							srcContainer.IP,
							dstContainer.IP,
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
