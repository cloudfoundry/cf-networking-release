package store

import (
	"policy-server/db"

	uuid "github.com/nu7hatch/gouuid"
)

type TerminalsTable struct {
}

func (e *TerminalsTable) Create(tx db.Transaction) (string, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	_, err = tx.Exec(tx.Rebind("INSERT INTO terminals (guid) VALUES (?)"), guid.String())
	if err != nil {
		return "", err
	}

	return guid.String(), nil
}

func (e *TerminalsTable) Delete(tx db.Transaction, terminalGUID string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM terminals WHERE guid = ?`), terminalGUID)
	return err
}
