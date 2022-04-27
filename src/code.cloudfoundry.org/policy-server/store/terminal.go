package store

import "code.cloudfoundry.org/cf-networking-helpers/db"

type TerminalsTable struct {
	Guids guidGenerator
}

func (t *TerminalsTable) Create(tx db.Transaction) (string, error) {
	guid := t.Guids.New()

	_, err := tx.Exec(tx.Rebind("INSERT INTO terminals (guid) VALUES (?)"), guid)
	if err != nil {
		return "", err
	}

	return guid, nil
}

func (t *TerminalsTable) Delete(tx db.Transaction, terminalGUID string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM terminals WHERE guid = ?`), terminalGUID)
	return err
}
