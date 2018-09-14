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
