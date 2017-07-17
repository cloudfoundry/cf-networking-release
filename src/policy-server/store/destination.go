package store

//go:generate counterfeiter -o fakes/destination_repo.go --fake-name DestinationRepo . DestinationRepo
type DestinationRepo interface {
	Create(Transaction, int, int, int, int, string) (int, error)
	Delete(Transaction, int) error
	GetID(Transaction, int, int, int, int, string) (int, error)
	CountWhereGroupID(Transaction, int) (int, error)
}

type DestinationTable struct {
}

func (d *DestinationTable) Create(tx Transaction, destination_group_id, port, startPort, endPort int, protocol string) (int, error) {
	_, err := tx.Exec(tx.Rebind(`
		INSERT INTO destinations (group_id, port, start_port, end_port, protocol)
		SELECT ?, ?, ?, ?, ?
		WHERE
		NOT EXISTS (
			SELECT *
			FROM destinations
			WHERE group_id = ? AND port = ? AND start_port = ? AND end_port = ? AND protocol = ?
		)`),
		destination_group_id,
		port,
		startPort,
		endPort,
		protocol,
		destination_group_id,
		port,
		startPort,
		endPort,
		protocol,
	)
	if err != nil {
		return -1, err
	}
	id, err := d.GetID(tx, destination_group_id, port, startPort, endPort, protocol)
	return id, err
}

func (d *DestinationTable) Delete(tx Transaction, id int) error {
	_, err := tx.Exec(
		tx.Rebind(`DELETE FROM destinations WHERE id = ?`),
		id,
	)
	return err
}

func (d *DestinationTable) GetID(tx Transaction, destination_group_id, port, startPort, endPort int, protocol string) (int, error) {
	var id int
	err := tx.QueryRow(tx.Rebind(`
		SELECT id FROM destinations
		WHERE group_id = ? AND port = ? AND start_port = ? AND end_port = ? AND protocol = ? FOR UPDATE`),
		destination_group_id,
		port,
		startPort,
		endPort,
		protocol,
	).Scan(&id)
	return id, err
}

func (d *DestinationTable) CountWhereGroupID(tx Transaction, group_id int) (int, error) {
	var count int
	err := tx.QueryRow(
		tx.Rebind(`SELECT COUNT(*) FROM destinations WHERE group_id = ?`),
		group_id,
	).Scan(&count)
	return count, err
}
