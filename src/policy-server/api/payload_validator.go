package api

import (
	"errors"
)

//go:generate counterfeiter -o fakes/payload_validator.go --fake-name PayloadValidator . payloadValidator
type payloadValidator interface {
	ValidatePayload(payload *PoliciesPayload) error
	ValidateEgressDestinationsPayload(payload *DestinationsPayload) error
}

type PayloadValidator struct {
	PolicyValidator validator
	EgressDestinationValidator egressDestinationsValidator
}

//go:generate counterfeiter -o fakes/egress_destinations_validator.go --fake-name EgressDestinationsValidator . egressDestinationsValidator
type egressDestinationsValidator interface {
	ValidateEgressDestinations([]EgressDestination) error
}

func (p *PayloadValidator) ValidatePayload(payload *PoliciesPayload) error {
	policiesEmpty := len(payload.Policies) == 0

	if policiesEmpty {
		return errors.New("expected policy")
	}

	if !policiesEmpty {
		err := p.PolicyValidator.ValidatePolicies(payload.Policies)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PayloadValidator) ValidateEgressDestinationsPayload(payload *DestinationsPayload) error {
	err := p.EgressDestinationValidator.ValidateEgressDestinations(payload.EgressDestinations)
	if err != nil {
		return err
	}
	return nil
}