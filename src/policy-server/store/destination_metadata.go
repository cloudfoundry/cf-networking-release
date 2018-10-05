package store

import (
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

type DestinationMetadataTable struct{}

func (d *DestinationMetadataTable) Create(tx db.Transaction, terminalGUID, name, description string) (int64, error) {
	driver := tx.DriverName()
	if driver == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO destination_metadatas (terminal_guid, name, description)
			VALUES (?,?,?)
		`),
			terminalGUID,
			name,
			description,
		)
		if err != nil {
			return -1, err
		}
		return result.LastInsertId()
	} else if driver == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO destination_metadatas (terminal_guid, name, description)
			VALUES (?,?,?)
			RETURNING id
		`),
			terminalGUID,
			name,
			description,
		).Scan(&id)

		if err != nil {
			return -1, err
		}

		return id, nil

	}
	return -1, fmt.Errorf("unknown driver: %s", driver)
}

func (d *DestinationMetadataTable) Update(tx db.Transaction, terminalGUID, name, description string) error {
	_, err := tx.Exec(tx.Rebind(`
		UPDATE destination_metadatas
		SET name = ?, description = ?
		WHERE terminal_guid = ?
	`),
		name,
		description,
		terminalGUID,
	)
	return err
}

func (d *DestinationMetadataTable) Delete(tx db.Transaction, guid string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM destination_metadatas WHERE terminal_guid = ?`), guid)
	return err
}
