package store

import (
	"code.cloudfoundry.org/cf-networking-helpers/db"
)

type DestinationMetadataTable struct{}

func (d *DestinationMetadataTable) Upsert(tx db.Transaction, terminalGUID, name, description string) error {
	var count int64
	err := tx.QueryRow(tx.Rebind(`
		SELECT COUNT(*) FROM destination_metadatas WHERE terminal_guid=?
	`), terminalGUID).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err := tx.Exec(tx.Rebind(`
			INSERT INTO destination_metadatas (terminal_guid, name, description)
			VALUES (?,?,?)
		`),
			terminalGUID,
			name,
			description,
		)
		return err
	} else {
		_, err = tx.Exec(tx.Rebind(`
			UPDATE destination_metadatas SET name=?, description=? WHERE terminal_guid=?
		`),
			name,
			description,
			terminalGUID,
		)
		return err
	}
}

func (d *DestinationMetadataTable) Delete(tx db.Transaction, guid string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM destination_metadatas WHERE terminal_guid=?`), guid)
	return err
}
