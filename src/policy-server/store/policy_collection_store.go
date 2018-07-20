package store

import (
	"fmt"
	"policy-server/db"
)

//go:generate counterfeiter -o fakes/egress_policy_store.go --fake-name EgressPolicyStore . egressPolicyStore
type egressPolicyStore interface {
	CreateWithTx(db.Transaction, []EgressPolicy) error
	DeleteWithTx(db.Transaction, []EgressPolicy) error
}

type PolicyCollectionStore struct {
	Conn              database
	PolicyStore       Store
	EgressPolicyStore egressPolicyStore
}

func (p *PolicyCollectionStore) Create(policyCollection PolicyCollection) error {
	tx, err := p.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}

	err = p.PolicyStore.CreateWithTx(tx, policyCollection.Policies) // TODO: Move rollback up to this level
	if err != nil {
		return rollback(tx, err)
	}

	err = p.EgressPolicyStore.CreateWithTx(tx, policyCollection.EgressPolicies)
	if err != nil {
		return rollback(tx, err)
	}

	return commit(tx)
}

func (p *PolicyCollectionStore) Delete(policyCollection PolicyCollection) error {
	tx, err := p.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}

	err = p.PolicyStore.DeleteWithTx(tx, policyCollection.Policies)
	if err != nil {
		return err
	}

	return commit(tx)
}
