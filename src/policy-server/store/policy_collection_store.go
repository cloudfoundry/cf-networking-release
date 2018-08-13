package store

import (
	"fmt"
	"policy-server/db"
)

//go:generate counterfeiter -o fakes/egress_policy_store.go --fake-name EgressPolicyStore . egressPolicyStore
type egressPolicyStore interface {
	CreateWithTx(db.Transaction, []EgressPolicy) error
	DeleteWithTx(db.Transaction, []EgressPolicy) error
	All() ([]EgressPolicy, error)
	ByGuids(srcGuids []string) ([]EgressPolicy, error)
}

type PolicyCollectionStore struct {
	Conn              Database
	PolicyStore       Store
	EgressPolicyStore egressPolicyStore
}

func (p *PolicyCollectionStore) Create(policyCollection PolicyCollection) error {
	tx, err := p.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}

	err = p.PolicyStore.CreateWithTx(tx, policyCollection.Policies)
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
		return rollback(tx, err)
	}

	err = p.EgressPolicyStore.DeleteWithTx(tx, policyCollection.EgressPolicies)
	if err != nil {
		return rollback(tx, err)
	}

	return commit(tx)
}

func (p *PolicyCollectionStore) All() (PolicyCollection, error) {
	c2cPolicies, err := p.PolicyStore.All()
	if err != nil {
		return PolicyCollection{}, err
	}

	egressPolicies, err := p.EgressPolicyStore.All()
	if err != nil {
		return PolicyCollection{}, err
	}

	return PolicyCollection{Policies: c2cPolicies, EgressPolicies: egressPolicies}, nil
}

func commit(tx db.Transaction) error {
	err := tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %s", err)
	}
	return nil
}

func rollback(tx db.Transaction, err error) error {
	txErr := tx.Rollback()
	if txErr != nil {
		return fmt.Errorf("database rollback: %s (sql error: %s)", txErr, err)
	}
	return err
}
