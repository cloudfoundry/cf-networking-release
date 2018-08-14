package api

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
)

//go:generate counterfeiter -o fakes/egress_validator.go --fake-name EgressValidator . egressValidator
type egressValidator interface {
	ValidateEgressPolicies(policies []EgressPolicy) error
}

//go:generate counterfeiter -o fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetLiveAppGUIDs(token string, appGUIDs []string) (map[string]struct{}, error)
}

//go:generate counterfeiter -o fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
}

type EgressValidator struct {
	CCClient  ccClient
	UAAClient uaaClient
}

func (v *EgressValidator) ValidateEgressPolicies(policies []EgressPolicy) error {
	for _, policy := range policies {
		if policy.Source == nil {
			return errors.New("missing egress source")
		}
		if policy.Source.ID == "" {
			return errors.New("missing egress source ID")
		}
		if policy.Source.Type != "" && policy.Source.Type != "app" && policy.Source.Type != "space" {
			return errors.New("source type must be app or space")
		}
		if policy.Destination == nil {
			return errors.New("missing egress destination")
		}
		if policy.Destination.Protocol == "" {
			return errors.New("missing egress destination protocol")
		}
		if len(policy.Destination.IPRanges) != 1 {
			return errors.New("expected exactly one iprange")
		}
		if policy.Destination.IPRanges[0].Start == "" {
			return errors.New("missing egress destination iprange start")
		}
		startIP := policy.Destination.IPRanges[0].Start
		parsedStartIP := net.ParseIP(startIP)
		if parsedStartIP == nil || parsedStartIP.To4() == nil {
			return fmt.Errorf("invalid ipv4 start ip address for ip range: %v", startIP)
		}
		endIP := policy.Destination.IPRanges[0].End
		parsedEndIP := net.ParseIP(endIP)
		if parsedEndIP == nil || parsedEndIP.To4() == nil {
			return fmt.Errorf("invalid ipv4 end ip address for ip range: %v", endIP)
		}

		if bytes.Compare(parsedStartIP, parsedEndIP) > 0 {
			return fmt.Errorf("start ip address should be before end ip address: start: %v end: %v", startIP, endIP)
		}

		if policy.Destination.Protocol != "icmp" && policy.Destination.Protocol != "tcp" && policy.Destination.Protocol != "udp" {
			return fmt.Errorf("protocol must be tcp, udp, or icmp")
		}

		if policy.Destination.Protocol == "icmp" {
			if policy.Destination.ICMPType == nil {
				return fmt.Errorf("missing icmp type")
			}
			if policy.Destination.ICMPCode == nil {
				return fmt.Errorf("missing icmp code")
			}
			if policy.Destination.Ports != nil {
				return fmt.Errorf("ports can not be defined with icmp")
			}
		}
	}

	token, err := v.UAAClient.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get uaa token: %s", err)
	}

	appGUIDSet := sourceAppGUIDs(policies)

	liveAppGUIDs, err := v.CCClient.GetLiveAppGUIDs(token, keys(appGUIDSet))
	if err != nil {
		return fmt.Errorf("failed to get live app guids: %s", err)
	}

	missingAppGUIDs := relativeComplement(appGUIDSet, liveAppGUIDs)

	if len(missingAppGUIDs) > 0 {
		return fmt.Errorf("app guids not found: [%s]", strings.Join(missingAppGUIDs, ", "))
	}

	return nil
}

func sourceAppGUIDs(policies []EgressPolicy) map[string]struct{} {
	appGUIDSet := make(map[string]struct{})
	for _, policy := range policies {
		if policy.Source.Type == "" || policy.Source.Type == "app" {
			appGUIDSet[policy.Source.ID] = struct{}{}
		}
	}
	return appGUIDSet
}

func keys(set map[string]struct{}) []string {
	var keys []string
	for key, _ := range set {
		keys = append(keys, key)
	}
	return keys
}

func relativeComplement(a map[string]struct{}, b map[string]struct{}) []string {
	result := []string{}
	for key, _ := range a {
		_, ok := b[key]
		if !ok {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
