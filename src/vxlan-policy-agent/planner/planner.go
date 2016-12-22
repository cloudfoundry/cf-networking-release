package planner

import (
	"lib/datastore"
	"lib/models"
	"lib/rules"
	"time"
	"vxlan-policy-agent/agent_metrics"
	"vxlan-policy-agent/enforcer"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPolicies() ([]models.Policy, error)
}

//go:generate counterfeiter -o ../fakes/dstore.go --fake-name Dstore . dstore
type dstore interface {
	ReadAll() (map[string]datastore.Container, error)
}

type VxlanPolicyPlanner struct {
	Logger            lager.Logger
	Datastore         dstore
	PolicyClient      policyClient
	VNI               int
	CollectionEmitter agent_metrics.TimeMetricsEmitter
	Chain             enforcer.Chain
	LoggingState      loggingStateGetter
}

type Container struct {
	Handle  string
	IP      string
	GroupID string
}

func (p *VxlanPolicyPlanner) getContainersMap(allContainers map[string]datastore.Container) (map[string][]string, error) {
	containers := map[string][]string{}
	for _, container := range allContainers {
		if container.Metadata == nil {
			continue
		}
		groupID, ok := container.Metadata["policy_group_id"].(string)
		if !ok {
			message := "Container metadata is missing key policy_group_id. CloudController version may be out of date or apps may need to be restaged."
			p.Logger.Debug("container-metadata-policy-group-id", lager.Data{"container_handle": container.Handle, "message": message})
			continue
		}
		containers[groupID] = append(containers[groupID], container.IP)
	}
	return containers, nil
}

func (p *VxlanPolicyPlanner) GetRulesAndChain() (enforcer.RulesWithChain, error) {
	containerMetadataStartTime := time.Now()
	containerMetadata, err := p.Datastore.ReadAll()
	if err != nil {
		p.Logger.Error("datastore", err)
		return enforcer.RulesWithChain{}, err
	}

	containers, err := p.getContainersMap(containerMetadata)
	if err != nil {
		p.Logger.Error("container-info", err)
		return enforcer.RulesWithChain{}, err
	}
	containerMetadataDuration := time.Now().Sub(containerMetadataStartTime)
	p.Logger.Debug("got-containers", lager.Data{"containers": containers})

	policyServerStartRequestTime := time.Now()
	policies, err := p.PolicyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("policy-client-get-policies", err)
		return enforcer.RulesWithChain{}, err
	}
	policyServerPollDuration := time.Now().Sub(policyServerStartRequestTime)
	p.CollectionEmitter.EmitAll(map[string]time.Duration{
		agent_metrics.MetricContainerMetadata: containerMetadataDuration,
		agent_metrics.MetricPolicyServerPoll:  policyServerPollDuration,
	})

	marksRuleset := []rules.IPTablesRule{}
	filterRuleset := []rules.IPTablesRule{}

	iptablesLoggingEnabled := p.LoggingState.IsEnabled()
	for _, policy := range policies {
		srcContainerIPs, srcOk := containers[policy.Source.ID]
		dstContainerIPs, dstOk := containers[policy.Destination.ID]

		if dstOk {
			// there are some containers on this host that are dests for the policy
			for _, dstContainerIP := range dstContainerIPs {
				if iptablesLoggingEnabled {
					filterRuleset = append(
						filterRuleset,
						rules.NewMarkLogRule(
							dstContainerIP,
							policy.Destination.Protocol,
							policy.Destination.Port,
							policy.Source.Tag,
							policy.Destination.ID,
						),
					)
				}
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
	return enforcer.RulesWithChain{
		Chain: p.Chain,
		Rules: ruleset,
	}, nil
}
