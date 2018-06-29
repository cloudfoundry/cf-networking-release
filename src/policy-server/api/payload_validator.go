package api

import (
	"errors"
)

//go:generate counterfeiter -o fakes/payload_validator.go --fake-name PayloadValidator . payloadValidator
type payloadValidator interface {
	ValidatePayload(payload *PoliciesPayload) error
}

type PayloadValidator struct {
	PolicyValidator       validator
	EgressPolicyValidator egressValidator
}

func (p *PayloadValidator) ValidatePayload(payload *PoliciesPayload) error {
	policiesEmpty := len(payload.Policies) == 0
	egressPoliciesEmpty := len(payload.EgressPolicies) == 0

	if policiesEmpty && egressPoliciesEmpty {
		return errors.New("expected policy or egress policy")
	}

	if !policiesEmpty {
		err := p.PolicyValidator.ValidatePolicies(payload.Policies)
		if err != nil {
			return err
		}
	}

	if !egressPoliciesEmpty {
		return p.EgressPolicyValidator.ValidateEgressPolicies(payload.EgressPolicies)
	}

	return nil
}
