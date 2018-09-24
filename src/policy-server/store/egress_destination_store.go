package store

import (
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

//go:generate counterfeiter -o fakes/egress_destination_repo.go --fake-name EgressDestinationRepo . egressDestinationRepo
type egressDestinationRepo interface {
	All(tx db.Transaction) ([]EgressDestination, error)
	CreateIPRange(tx db.Transaction, destinationTerminalGUID, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) (int64, error)
}

//go:generate counterfeiter -o fakes/destination_metadata_repo.go --fake-name DestinationMetadataRepo . destinationMetadataRepo
type destinationMetadataRepo interface {
	Create(tx db.Transaction, terminalGUID, name, description string) (int64, error)
}

type EgressDestinationStore struct {
	Conn                    Database
	EgressDestinationRepo   egressDestinationRepo
	TerminalsRepo           terminalsRepo
	DestinationMetadataRepo destinationMetadataRepo
}

func (e *EgressDestinationStore) All() ([]EgressDestination, error) {
	tx, err := e.Conn.Beginx()
	if err != nil {
		return []EgressDestination{}, fmt.Errorf("egress destination store create transaction: %s", err)
	}
	defer tx.Rollback()
	return e.EgressDestinationRepo.All(tx)
}

func (e *EgressDestinationStore) Create(egressDestinations []EgressDestination) ([]EgressDestination, error) {
	tx, err := e.Conn.Beginx()
	if err != nil {
		return []EgressDestination{}, fmt.Errorf("egress destination store create transaction: %s", err)
	}

	results := []EgressDestination{}
	for _, egressDestination := range egressDestinations {
		destinationTerminalGUID, err := e.TerminalsRepo.Create(tx)
		if err != nil {
			tx.Rollback()
			return []EgressDestination{}, fmt.Errorf("egress destination store create terminal: %s", err)
		}

		_, err = e.DestinationMetadataRepo.Create(tx, destinationTerminalGUID, egressDestination.Name, egressDestination.Description)
		if err != nil {
			tx.Rollback()
			return []EgressDestination{}, fmt.Errorf("egress destination store create destination metadata: %s", err)
		}

		var startPort, endPort int64
		if len(egressDestination.Ports) > 0 {
			startPort = int64(egressDestination.Ports[0].Start)
			endPort = int64(egressDestination.Ports[0].End)
		}

		_, err = e.EgressDestinationRepo.CreateIPRange(
			tx,
			destinationTerminalGUID,
			egressDestination.IPRanges[0].Start,
			egressDestination.IPRanges[0].End,
			egressDestination.Protocol,
			startPort,
			endPort,
			int64(egressDestination.ICMPType),
			int64(egressDestination.ICMPCode),
		)
		if err != nil {
			tx.Rollback()
			return []EgressDestination{}, fmt.Errorf("egress destination store create ip range: %s", err)
		}

		egressDestination.GUID = destinationTerminalGUID
		results = append(results, egressDestination)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return []EgressDestination{}, fmt.Errorf("egress destination store commit transaction: %s", err)
	}

	return results, nil
}
