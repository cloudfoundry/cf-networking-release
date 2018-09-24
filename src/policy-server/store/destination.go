package store

import "code.cloudfoundry.org/cf-networking-helpers/db"

//go:generate counterfeiter -o fakes/destination_repo.go --fake-name DestinationRepo . DestinationRepo
type DestinationRepo interface {
	Create(db.Transaction, int, int, int, int, string) (int, error)
	Delete(db.Transaction, int) error
	GetID(db.Transaction, int, int, int, int, string) (int, error)
	CountWhereGroupID(db.Transaction, int) (int, error)
}

type DestinationTable struct {
}

func (d *DestinationTable) Create(tx db.Transaction, destinationGroupId, port, startPort, endPort int, protocol string) (int, error) {
	dualStatement := ""
	if tx.DriverName() == "mysql" {
		dualStatement = " FROM DUAL "
	}

	_, err := tx.Exec(tx.Rebind(`
		INSERT INTO destinations (group_id, port, start_port, end_port, protocol)
		SELECT ?, ?, ?, ?, ? `+dualStatement+`
		WHERE
		NOT EXISTS (
			SELECT *
			FROM destinations
			WHERE group_id = ? AND port = ? AND start_port = ? AND end_port = ? AND protocol = ?
		)`),
		destinationGroupId,
		port,
		startPort,
		endPort,
		protocol,
		destinationGroupId,
		port,
		startPort,
		endPort,
		protocol,
	)
	if err != nil {
		return -1, err
	}
	id, err := d.GetID(tx, destinationGroupId, port, startPort, endPort, protocol)
	return id, err
}

func (d *DestinationTable) Delete(tx db.Transaction, id int) error {
	_, err := tx.Exec(
		tx.Rebind(`DELETE FROM destinations WHERE id = ?`),
		id,
	)
	return err
}

func (d *DestinationTable) GetID(tx db.Transaction, destinationGroupId, port, startPort, endPort int, protocol string) (int, error) {
	var id int
	lockStatement := " FOR UPDATE "
	if tx.DriverName() == "mysql" {
		lockStatement = " LOCK IN SHARE MODE "
	}
	err := tx.QueryRow(tx.Rebind(`
		SELECT id FROM destinations
		WHERE group_id = ? AND port = ? AND start_port = ? AND end_port = ? AND protocol = ? `+lockStatement),
		destinationGroupId,
		port,
		startPort,
		endPort,
		protocol,
	).Scan(&id)
	return id, err
}

func (d *DestinationTable) CountWhereGroupID(tx db.Transaction, groupId int) (int, error) {
	var count int
	err := tx.QueryRow(
		tx.Rebind(`SELECT COUNT(*) FROM destinations WHERE group_id = ?`),
		groupId,
	).Scan(&count)
	return count, err
}
