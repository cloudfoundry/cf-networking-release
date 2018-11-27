package store

import (
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

//go:generate counterfeiter -o fakes/egress_policy_repo.go --fake-name EgressPolicyRepo . egressPolicyRepo
type egressPolicyRepo interface {
	CreateApp(tx db.Transaction, sourceTerminalGUID string, appGUID string) (int64, error)
	CreateIPRange(tx db.Transaction, destinationTerminalGUID string, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) (int64, error)
	CreateEgressPolicy(tx db.Transaction, sourceTerminalGUID, destinationTerminalGUID, appLifecycle string) (string, error)
	CreateSpace(tx db.Transaction, sourceTerminalGUID string, spaceGUID string) (int64, error)
	GetTerminalByAppGUID(tx db.Transaction, appGUID string) (string, error)
	GetTerminalBySpaceGUID(tx db.Transaction, appGUID string) (string, error)
	GetAllPolicies() ([]EgressPolicy, error)
	GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames []string) ([]EgressPolicy, error)
	GetBySourceGuids(ids []string) ([]EgressPolicy, error)
	GetByGUID(tx db.Transaction, ids ...string) ([]EgressPolicy, error)
	DeleteEgressPolicy(tx db.Transaction, egressPolicyGUID string) error
	DeleteIPRange(tx db.Transaction, ipRangeID int64) error
	DeleteApp(tx db.Transaction, terminalID string) error
	DeleteSpace(tx db.Transaction, spaceID string) error
	IsTerminalInUse(tx db.Transaction, terminalGUID string) (bool, error)
}

//go:generate counterfeiter -o fakes/terminals_repo.go --fake-name TerminalsRepo . terminalsRepo
type terminalsRepo interface {
	Create(tx db.Transaction) (string, error)
	Delete(tx db.Transaction, terminalGUID string) error
}

type EgressPolicyStore struct {
	TerminalsRepo    terminalsRepo
	EgressPolicyRepo egressPolicyRepo
	Conn             Database
}

func (e *EgressPolicyStore) Create(policies []EgressPolicy) ([]EgressPolicy, error) {
	tx, err := e.Conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("create transaction: %s", err)
	}

	policies, err = e.createWithTx(tx, policies)
	if err != nil {
		return nil, rollback(tx, err)
	}

	return policies, commit(tx)
}

func (e *EgressPolicyStore) createWithTx(tx db.Transaction, policies []EgressPolicy) ([]EgressPolicy, error) {
	var createdPolicies []EgressPolicy
	for _, policy := range policies {
		var sourceTerminalGUID string
		var err error

		matchingPolicy, err := e.GetByFilter([]string{policy.Source.ID}, []string{policy.Source.Type}, []string{policy.Destination.GUID}, []string{})
		if err != nil {
			return nil, fmt.Errorf("failed to filter existing policies: %s", err)
		}
		if len(matchingPolicy) > 0 {
			createdPolicies = append(createdPolicies, matchingPolicy[0])
			continue
		}

		switch policy.Source.Type {
		case "space":
			sourceTerminalGUID, err = e.EgressPolicyRepo.GetTerminalBySpaceGUID(tx, policy.Source.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get terminal by space guid: %s", err)
			}

			if sourceTerminalGUID == "" {
				sourceTerminalGUID, err = e.TerminalsRepo.Create(tx)
				if err != nil {
					return nil, fmt.Errorf("failed to create source terminal: %s", err)
				}

				_, err = e.EgressPolicyRepo.CreateSpace(tx, sourceTerminalGUID, policy.Source.ID)
				if err != nil {
					return nil, fmt.Errorf("failed to create space: %s", err)
				}
			}
		default:
			sourceTerminalGUID, err = e.EgressPolicyRepo.GetTerminalByAppGUID(tx, policy.Source.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get terminal by app guid: %s", err)
			}

			if sourceTerminalGUID == "" {
				sourceTerminalGUID, err = e.TerminalsRepo.Create(tx)
				if err != nil {
					return nil, fmt.Errorf("failed to create source terminal: %s", err)
				}

				_, err = e.EgressPolicyRepo.CreateApp(tx, sourceTerminalGUID, policy.Source.ID)
				if err != nil {
					return nil, fmt.Errorf("failed to create source app: %s", err)
				}
			}
		}

		createdPolicyGUID, err := e.EgressPolicyRepo.CreateEgressPolicy(tx, sourceTerminalGUID, policy.Destination.GUID, policy.AppLifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to create egress policy: %s", err)
		}

		policy.ID = createdPolicyGUID
		policy.Source.TerminalGUID = sourceTerminalGUID

		createdPolicies = append(createdPolicies, policy)
	}
	return createdPolicies, nil
}

func (e *EgressPolicyStore) Delete(egressPolicyGUIDs ...string) ([]EgressPolicy, error) {
	tx, err := e.Conn.Beginx()
	if err != nil {
		return []EgressPolicy{}, fmt.Errorf("create transaction: %s", err)
	}

	egressPolicies, err := e.deleteWithTx(tx, egressPolicyGUIDs...)
	if err != nil {
		return []EgressPolicy{}, rollback(tx, err)
	}

	return egressPolicies, commit(tx)
}

func (e *EgressPolicyStore) deleteWithTx(tx db.Transaction, egressPolicyGUIDs ...string) ([]EgressPolicy, error) {
	egressPolicies, err := e.EgressPolicyRepo.GetByGUID(tx, egressPolicyGUIDs...)
	if err != nil {
		return []EgressPolicy{}, fmt.Errorf("failed to find egress policy: %s", err)
	}

	if len(egressPolicies) == 0 {
		return egressPolicies, nil
	}

	for _, egressPolicy := range egressPolicies {
		err = e.EgressPolicyRepo.DeleteEgressPolicy(tx, egressPolicy.ID)
		if err != nil {
			return []EgressPolicy{}, fmt.Errorf("failed to delete egress policy: %s", err)
		}

		terminalInUse, err := e.EgressPolicyRepo.IsTerminalInUse(tx, egressPolicy.Source.TerminalGUID)
		if err != nil {
			return []EgressPolicy{}, fmt.Errorf("failed to check if source terminal is in use: %s", err)
		}

		if !terminalInUse {
			if egressPolicy.Source.Type == "app" {
				err = e.EgressPolicyRepo.DeleteApp(tx, egressPolicy.Source.TerminalGUID)
				if err != nil {
					return []EgressPolicy{}, fmt.Errorf("failed to delete source app: %s", err)
				}
			}

			if egressPolicy.Source.Type == "space" {
				err = e.EgressPolicyRepo.DeleteSpace(tx, egressPolicy.Source.TerminalGUID)
				if err != nil {
					return []EgressPolicy{}, fmt.Errorf("failed to delete source space: %s", err)
				}
			}

			err = e.TerminalsRepo.Delete(tx, egressPolicy.Source.TerminalGUID)
			if err != nil {
				return []EgressPolicy{}, fmt.Errorf("failed to delete source terminal: %s", err)
			}
		}
	}

	return egressPolicies, nil
}

func (e *EgressPolicyStore) All() ([]EgressPolicy, error) {
	return e.EgressPolicyRepo.GetAllPolicies()
}

func (e *EgressPolicyStore) GetBySourceGuids(ids []string) ([]EgressPolicy, error) {
	policies, err := e.EgressPolicyRepo.GetBySourceGuids(ids)
	if err != nil {
		return []EgressPolicy{}, fmt.Errorf("failed to get policies by guids: %s", err)
	}
	return policies, nil
}

func (e *EgressPolicyStore) GetByFilter(sourceId, sourceType, destinationId, destinationName []string) ([]EgressPolicy, error) {
	policies, err := e.EgressPolicyRepo.GetByFilter(sourceId, sourceType, destinationId, destinationName)
	if err != nil {
		return []EgressPolicy{}, fmt.Errorf("failed to get policies by filter: %s", err)
	}
	return policies, nil
}
