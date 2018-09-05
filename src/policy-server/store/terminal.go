package store

import (
	"fmt"
	"policy-server/db"
)

type TerminalsTable struct {
}

func (e *TerminalsTable) Create(tx db.Transaction) (int64, error) {
	driverName := tx.DriverName()

	if driverName == "mysql" {
		result, err := tx.Exec("INSERT INTO terminals (id) VALUES (NULL)")
		if err != nil {
			return -1, err
		}

		return result.LastInsertId()

	} else if driverName == "postgres" {
		var id int64
		err := tx.QueryRow("INSERT INTO terminals default values RETURNING id").Scan(&id)
		if err != nil {
			return -1, err
		}

		return id, nil
	}

	return -1, fmt.Errorf("unknown driver: %s", driverName)
}

func (e *TerminalsTable) Delete(tx db.Transaction, terminalID int64) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM terminals WHERE id = ?`), terminalID)
	return err
}
