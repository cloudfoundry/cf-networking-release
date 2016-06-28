package handlers

import (
	"errors"
	"policy-server/models"
)

func validateFields(policies []models.Policy) error {
	for _, policy := range policies {
		if policy.Source.ID == "" {
			return errors.New("missing source id")
		}
		if policy.Destination.ID == "" {
			return errors.New("missing destination id")
		}
		if policy.Destination.Protocol == "" {
			return errors.New("missing destination protocol")
		}
		if policy.Destination.Port == 0 {
			return errors.New("missing destination port")
		}

		if policy.Source.Tag != "" || policy.Destination.Tag != "" {
			return errors.New("tags may not be specified")
		}
	}
	return nil
}
