package store

import (
	"fmt"
	"policy-server/db"
)

//go:generate counterfeiter -o fakes/egress_policy_repo.go --fake-name EgressPolicyRepo . egressPolicyRepo
type egressPolicyRepo interface {
	CreateTerminal(tx db.Transaction) (int64, error)
	CreateApp(tx db.Transaction, sourceTerminalID int64, appGUID string) (int64, error)
	CreateIPRange(tx db.Transaction, destinationTerminalID int64, startIP, endIP, protocol string) (int64, error)
	CreateEgressPolicy(tx db.Transaction, sourceTerminalID, destinationTerminalID int64) (int64, error)
	GetTerminalByAppGUID(tx db.Transaction, appGUID string) (int64, error)
	GetAllPolicies() ([]EgressPolicy, error)
	GetByGuids(ids []string) ([]EgressPolicy, error)
}

type EgressPolicyStore struct {
	EgressPolicyRepo egressPolicyRepo
}

func (e *EgressPolicyStore) CreateWithTx(tx db.Transaction, policies []EgressPolicy) error {
	for _, policy := range policies {
		sourceTerminalID, err := e.EgressPolicyRepo.GetTerminalByAppGUID(tx, policy.Source.ID)
		if err != nil {
			return fmt.Errorf("failed to get terminal by app guid: %s", err)
		}

		if sourceTerminalID == -1 {
			sourceTerminalID, err = e.EgressPolicyRepo.CreateTerminal(tx)
			if err != nil {
				return fmt.Errorf("failed to create source terminal: %s", err)
			}

			_, err = e.EgressPolicyRepo.CreateApp(tx, sourceTerminalID, policy.Source.ID)
			if err != nil {
				return fmt.Errorf("failed to create source app: %s", err)
			}
		}

		destinationTerminalID, err := e.EgressPolicyRepo.CreateTerminal(tx)
		if err != nil {
			return fmt.Errorf("failed to create destination terminal: %s", err)
		}

		_, err = e.EgressPolicyRepo.CreateIPRange(
			tx,
			destinationTerminalID,
			policy.Destination.IPRanges[0].Start,
			policy.Destination.IPRanges[0].End,
			policy.Destination.Protocol)
		if err != nil {
			return fmt.Errorf("failed to create ip range: %s", err)
		}

		_, err = e.EgressPolicyRepo.CreateEgressPolicy(tx, sourceTerminalID, destinationTerminalID)
		if err != nil {
			return fmt.Errorf("failed to create egress policy: %s", err)
		}
	}
	return nil
}

func (e *EgressPolicyStore) DeleteWithTx(_ db.Transaction, _ []EgressPolicy) error {
	panic("not implemented")
}

func (e *EgressPolicyStore) All() ([]EgressPolicy, error) {
	return e.EgressPolicyRepo.GetAllPolicies()
}

func (e *EgressPolicyStore) ByGuids(ids []string) ([]EgressPolicy, error) {
	policies, err := e.EgressPolicyRepo.GetByGuids(ids)
	if err != nil {
		return []EgressPolicy{}, fmt.Errorf("failed to get policies by guids: %s", err)
	}
	return policies, nil
}
