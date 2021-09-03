package api

import (
	"errors"
	"fmt"
)

//go:generate counterfeiter -o fakes/policy_validator.go --fake-name PolicyValidator . policyValidator
type policyValidator interface {
	ValidatePolicies(policies []Policy) error
}

type PolicyValidator struct{}

func (v *PolicyValidator) ValidatePolicies(policies []Policy) error {
	if len(policies) == 0 {
		return errors.New("missing policies")
	}

	for _, policy := range policies {
		if policy.Source.ID == "" {
			return errors.New("missing source id")
		}

		if policy.Destination.ID == "" {
			return errors.New("missing destination id")
		}

		if policy.Destination.Protocol != "udp" && policy.Destination.Protocol != "tcp" {
			return errors.New("invalid destination protocol, specify either udp or tcp")
		}

		if policy.Destination.Ports.Start > policy.Destination.Ports.End {
			return fmt.Errorf("invalid port range %d-%d, start must be less than or equal to end", policy.Destination.Ports.Start, policy.Destination.Ports.End)
		}

		if policy.Destination.Ports.Start < 0 {
			return fmt.Errorf("invalid start port %d, must be in range 1-65535", policy.Destination.Ports.Start)
		}

		if policy.Destination.Ports.Start == 0 {
			return fmt.Errorf("missing start port")
		}

		if policy.Destination.Ports.End > 65535 {
			return fmt.Errorf("invalid end port %d, must be in range 1-65535", policy.Destination.Ports.End)
		}

		if policy.Source.Tag != "" || policy.Destination.Tag != "" {
			return errors.New("tags may not be specified")
		}
	}
	return nil
}
