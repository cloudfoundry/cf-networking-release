package api

import (
	"errors"
	"fmt"
	"net"
)

//go:generate counterfeiter -o fakes/egress_validator.go --fake-name EgressValidator . egressValidator
type egressValidator interface {
	ValidateEgressPolicies(policies []EgressPolicy) error
}

type EgressValidator struct{}

func (v *EgressValidator) ValidateEgressPolicies(policies []EgressPolicy) error {
	if len(policies) != 1 {
		return errors.New("expected exactly one egress policy")
	}

	if policies[0].Source == nil {
		return errors.New("missing egress source")
	}

	if policies[0].Source.ID == "" {
		return errors.New("missing egress source ID")
	}

	if policies[0].Destination == nil {
		return errors.New("missing egress destination")
	}

	if policies[0].Destination.Protocol == "" {
		return errors.New("missing egress destination protocol")
	}

	if len(policies[0].Destination.IPRanges) != 1 {
		return errors.New("expected exactly one iprange")
	}

	if policies[0].Destination.IPRanges[0].Start == "" {
		return errors.New("missing egress destination iprange start")
	}

	startIP := policies[0].Destination.IPRanges[0].Start
	parsedStartIP := net.ParseIP(startIP)
	if parsedStartIP == nil || parsedStartIP.To4() == nil {
		return fmt.Errorf("invalid ipv4 start ip address for ip range: %v", startIP)
	}

	endIP := policies[0].Destination.IPRanges[0].End
	parsedEndIP := net.ParseIP(endIP)
	if parsedEndIP == nil || parsedEndIP.To4() == nil {
		return fmt.Errorf("invalid ipv4 end ip address for ip range: %v", endIP)
	}

	return nil
}
