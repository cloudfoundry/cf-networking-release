package store

import "fmt"

type PolicyCollectionStore struct {
	Conn        database
	PolicyStore Store
}

func (p *PolicyCollectionStore) Create(policyCollection PolicyCollection) error {
	tx, err := p.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}

	err = p.PolicyStore.CreateWithTx(tx, policyCollection.Policies)
	if err != nil {
		return err
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
