package store

import (
	"policy-server/db"
	"fmt"
)

//go:generate counterfeiter -o fakes/egress_destination_repo.go --fake-name EgressDestinationRepo . egressDestinationRepo
type egressDestinationRepo interface {
	All(tx db.Transaction) ([]EgressDestination, error)
}

type EgressDestinationStore struct {
	Conn                  Database
	EgressDestinationRepo egressDestinationRepo
}

func (e *EgressDestinationStore) All() ([]EgressDestination, error) {
	tx, err := e.Conn.Beginx()
	if err != nil {
		return []EgressDestination{}, fmt.Errorf("egress destination store create transaction: %s", err)
	}
	defer tx.Rollback()
	return e.EgressDestinationRepo.All(tx)
}