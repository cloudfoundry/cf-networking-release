package planner

import (
	"lib/datastore"
	"lib/rules"
	"policy-server/api"
	"sort"
	"time"
	"vxlan-policy-agent/enforcer"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPoliciesByID(ids ...string) ([]api.Policy, error)
}

//go:generate counterfeiter -o fakes/dstore.go --fake-name Dstore . dstore
type dstore interface {
	ReadAll() (map[string]datastore.Container, error)
}

//go:generate counterfeiter -o fakes/metrics_sender.go --fake-name MetricsSender . metricsSender
type metricsSender interface {
	SendDuration(string, time.Duration)
}

//go:generate counterfeiter -o fakes/loggingStateGetter.go --fake-name LoggingStateGetter . loggingStateGetter
type loggingStateGetter interface {
	IsEnabled() bool
}

type VxlanPolicyPlanner struct {
	Logger                        lager.Logger
	Datastore                     dstore
	PolicyClient                  policyClient
	VNI                           int
	MetricsSender                 metricsSender
	Chain                         enforcer.Chain
	LoggingState                  loggingStateGetter
	IPTablesAcceptedUDPLogsPerSec int
}

type Container struct {
	Handle  string
	IP      string
	GroupID string
}

const metricContainerMetadata = "containerMetadataTime"
const metricPolicyServerPoll = "policyServerPollTime"

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
	groupIDs := make([]string, len(containers))
	i := 0
	for groupID := range containers {
		groupIDs[i] = groupID
		i++
	}
	if err != nil {
		p.Logger.Error("container-info", err)
		return enforcer.RulesWithChain{}, err
	}
	containerMetadataDuration := time.Now().Sub(containerMetadataStartTime)
	p.Logger.Debug("got-containers", lager.Data{"containers": containers})

	policyServerStartRequestTime := time.Now()
	var policies []api.Policy
	if len(groupIDs) > 0 {
		policies, err = p.PolicyClient.GetPoliciesByID(groupIDs...)
		if err != nil {
			p.Logger.Error("policy-client-get-policies", err)
			return enforcer.RulesWithChain{}, err
		}
	}

	policyServerPollDuration := time.Now().Sub(policyServerStartRequestTime)
	p.MetricsSender.SendDuration(metricContainerMetadata, containerMetadataDuration)
	p.MetricsSender.SendDuration(metricPolicyServerPoll, policyServerPollDuration)

	marksRuleset := []rules.IPTablesRule{}
	markedSourceIPs := make(map[string]struct{})
	filterRuleset := []rules.IPTablesRule{}

	iptablesLoggingEnabled := p.LoggingState.IsEnabled()
	policySlice := api.PolicySlice(policies)
	sort.Sort(policySlice)
	for _, policy := range policySlice {
		srcContainerIPs, srcOk := containers[policy.Source.ID]
		dstContainerIPs, dstOk := containers[policy.Destination.ID]

		if dstOk {
			// there are some containers on this host that are dests for the policy
			ips := sort.StringSlice(dstContainerIPs)
			sort.Sort(ips)
			for _, dstContainerIP := range ips {
				if iptablesLoggingEnabled {
					filterRuleset = append(
						filterRuleset,
						rules.NewMarkAllowLogRule(
							dstContainerIP,
							policy.Destination.Protocol,
							policy.Destination.Ports.Start,
							policy.Destination.Ports.End,
							policy.Source.Tag,
							policy.Destination.ID,
							p.IPTablesAcceptedUDPLogsPerSec,
						),
					)
				}
				filterRuleset = append(
					filterRuleset,
					rules.NewMarkAllowRule(
						dstContainerIP,
						policy.Destination.Protocol,
						policy.Destination.Ports.Start,
						policy.Destination.Ports.End,
						policy.Source.Tag,
						policy.Source.ID,
						policy.Destination.ID,
					),
				)
			}
		}

		if srcOk {
			// there are some containers on this host that are sources for the policy
			ips := sort.StringSlice(srcContainerIPs)
			sort.Sort(ips)
			for _, srcContainerIP := range ips {
				_, added := markedSourceIPs[srcContainerIP]
				if !added {
					rule := rules.NewMarkSetRule(srcContainerIP, policy.Source.Tag, policy.Source.ID)
					marksRuleset = append(marksRuleset, rule)
					markedSourceIPs[srcContainerIP] = struct{}{}
				}
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
