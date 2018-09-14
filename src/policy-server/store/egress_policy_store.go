package store

import (
	"fmt"
	"policy-server/db"
)

//go:generate counterfeiter -o fakes/egress_policy_repo.go --fake-name EgressPolicyRepo . egressPolicyRepo
type egressPolicyRepo interface {
	CreateApp(tx db.Transaction, sourceTerminalGUID string, appGUID string) (int64, error)
	CreateIPRange(tx db.Transaction, destinationTerminalGUID string, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) (int64, error)
	CreateEgressPolicy(tx db.Transaction, sourceTerminalGUID, destinationTerminalGUID string) (string, error)
	CreateSpace(tx db.Transaction, sourceTerminalGUID string, spaceGUID string) (int64, error)
	GetTerminalByAppGUID(tx db.Transaction, appGUID string) (string, error)
	GetTerminalBySpaceGUID(tx db.Transaction, appGUID string) (string, error)
	GetAllPolicies() ([]EgressPolicy, error)
	GetByGuids(ids []string) ([]EgressPolicy, error)
	GetIDCollectionsByEgressPolicy(tx db.Transaction, egressPolicy EgressPolicy) ([]EgressPolicyIDCollection, error)
	DeleteEgressPolicy(tx db.Transaction, egressPolicyGUID string) error
	DeleteIPRange(tx db.Transaction, ipRangeID int64) error
	DeleteApp(tx db.Transaction, appID int64) error
	DeleteSpace(tx db.Transaction, spaceID int64) error
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

func (e *EgressPolicyStore) Create(policies []EgressPolicy) error {
	tx, err := e.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}

	err = e.createWithTx(tx, policies)
	if err != nil {
		return rollback(tx, err)
	}

	return commit(tx)
}

func (e *EgressPolicyStore) createWithTx(tx db.Transaction, policies []EgressPolicy) error {
	for _, policy := range policies {

		ids, err := e.EgressPolicyRepo.GetIDCollectionsByEgressPolicy(tx, policy)
		if err != nil {
			return err
		}

		if len(ids) > 0 {
			continue
		}

		var sourceTerminalGUID string

		switch policy.Source.Type {
		case "space":
			sourceTerminalGUID, err = e.EgressPolicyRepo.GetTerminalBySpaceGUID(tx, policy.Source.ID)
			if err != nil {
				return fmt.Errorf("failed to get terminal by space guid: %s", err)
			}

			if sourceTerminalGUID == "" {
				sourceTerminalGUID, err = e.TerminalsRepo.Create(tx)
				if err != nil {
					return fmt.Errorf("failed to create source terminal: %s", err)
				}

				_, err = e.EgressPolicyRepo.CreateSpace(tx, sourceTerminalGUID, policy.Source.ID)
				if err != nil {
					return fmt.Errorf("failed to create space: %s", err)
				}
			}
		default:
			sourceTerminalGUID, err = e.EgressPolicyRepo.GetTerminalByAppGUID(tx, policy.Source.ID)
			if err != nil {
				return fmt.Errorf("failed to get terminal by app guid: %s", err)
			}

			if sourceTerminalGUID == "" {
				sourceTerminalGUID, err = e.TerminalsRepo.Create(tx)
				if err != nil {
					return fmt.Errorf("failed to create source terminal: %s", err)
				}

				_, err = e.EgressPolicyRepo.CreateApp(tx, sourceTerminalGUID, policy.Source.ID)
				if err != nil {
					return fmt.Errorf("failed to create source app: %s", err)
				}
			}
		}

		destinationTerminalGUID, err := e.TerminalsRepo.Create(tx)
		if err != nil {
			return fmt.Errorf("failed to create destination terminal: %s", err)
		}

		var startPort, endPort int64
		if len(policy.Destination.Ports) > 0 {
			startPort = int64(policy.Destination.Ports[0].Start)
			endPort = int64(policy.Destination.Ports[0].End)
		}

		_, err = e.EgressPolicyRepo.CreateIPRange(
			tx,
			destinationTerminalGUID,
			policy.Destination.IPRanges[0].Start,
			policy.Destination.IPRanges[0].End,
			policy.Destination.Protocol,
			startPort,
			endPort,
			int64(policy.Destination.ICMPType),
			int64(policy.Destination.ICMPCode),
		)
		if err != nil {
			return fmt.Errorf("failed to create ip range: %s", err)
		}

		_, err = e.EgressPolicyRepo.CreateEgressPolicy(tx, sourceTerminalGUID, destinationTerminalGUID)
		if err != nil {
			return fmt.Errorf("failed to create egress policy: %s", err)
		}
	}
	return nil
}

func (e *EgressPolicyStore) Delete(egressPolicies []EgressPolicy) error {
	tx, err := e.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}

	err = e.deleteWithTx(tx, egressPolicies)
	if err != nil {
		return rollback(tx, err)
	}

	return commit(tx)
}

func (e *EgressPolicyStore) deleteWithTx(tx db.Transaction, egressPolicies []EgressPolicy) error {
	for _, policy := range egressPolicies {
		egressPolicyIDCollections, err := e.EgressPolicyRepo.GetIDCollectionsByEgressPolicy(tx, policy)
		if err != nil {
			return fmt.Errorf("failed to find egress policy: %s", err)
		}

		for _, egressPolicyIDCollection := range egressPolicyIDCollections {
			err = e.EgressPolicyRepo.DeleteEgressPolicy(tx, egressPolicyIDCollection.EgressPolicyGUID)
			if err != nil {
				return fmt.Errorf("failed to delete egress policy: %s", err)
			}

			err = e.EgressPolicyRepo.DeleteIPRange(tx, egressPolicyIDCollection.DestinationIPRangeID)
			if err != nil {
				return fmt.Errorf("failed to delete destination ip range: %s", err)
			}

			err = e.TerminalsRepo.Delete(tx, egressPolicyIDCollection.DestinationTerminalGUID)
			if err != nil {
				return fmt.Errorf("failed to delete destination terminal: %s", err)
			}

			terminalInUse, err := e.EgressPolicyRepo.IsTerminalInUse(tx, egressPolicyIDCollection.SourceTerminalGUID)
			if err != nil {
				return fmt.Errorf("failed to check if source terminal is in use: %s", err)
			}

			if !terminalInUse {
				if egressPolicyIDCollection.SourceAppID != -1 {
					err = e.EgressPolicyRepo.DeleteApp(tx, egressPolicyIDCollection.SourceAppID)
					if err != nil {
						return fmt.Errorf("failed to delete source app: %s", err)
					}
				}

				if egressPolicyIDCollection.SourceSpaceID != -1 {
					err = e.EgressPolicyRepo.DeleteSpace(tx, egressPolicyIDCollection.SourceSpaceID)
					if err != nil {
						return fmt.Errorf("failed to delete source space: %s", err)
					}
				}

				err = e.TerminalsRepo.Delete(tx, egressPolicyIDCollection.SourceTerminalGUID)
				if err != nil {
					return fmt.Errorf("failed to delete source terminal: %s", err)
				}
			}
		}
	}

	return nil
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
