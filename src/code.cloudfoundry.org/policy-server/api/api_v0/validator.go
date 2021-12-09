package api_v0

import (
	"errors"
	"fmt"
)

//counterfeiter:generate -o fakes/validator.go --fake-name PolicyValidator . validator
type validator interface {
	ValidatePolicies(policies []Policy) error
}

type Validator struct{}

func (v *Validator) ValidatePolicies(policies []Policy) error {
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
		if policy.Destination.Port < 0 {
			return fmt.Errorf("invalid port %d, must be in range 1-65535", policy.Destination.Port)
		}
		if policy.Destination.Port == 0 {
			return fmt.Errorf("missing port")
		}
		if policy.Source.Tag != "" || policy.Destination.Tag != "" {
			return errors.New("tags may not be specified")
		}
	}
	return nil
}
