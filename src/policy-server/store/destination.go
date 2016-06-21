package store

//go:generate counterfeiter -o ../fakes/destination_repo.go --fake-name DestinationRepo . DestinationRepo
type DestinationRepo interface {
	Create(Transaction, int, int, string) (int, error)
	GetID(Transaction, int, int, string) (int, error)
}

type Destination struct {
}

func (d *Destination) Create(tx Transaction, destination_group_id int, port int, protocol string) (int, error) {
	_, err := tx.Exec(tx.Rebind(`
		INSERT INTO destinations (group_id, port, protocol)
		SELECT ?, ?, ?
		WHERE
		NOT EXISTS (
			SELECT *
			FROM destinations
			WHERE group_id = ? AND port = ? AND protocol = ?
		)`),
		destination_group_id,
		port,
		protocol,
		destination_group_id,
		port,
		protocol,
	)
	if err != nil {
		return -1, err
	}

	return d.GetID(tx, destination_group_id, port, protocol)
}
func (d *Destination) GetID(tx Transaction, destination_group_id int, port int, protocol string) (int, error) {
	var id int
	err := tx.QueryRow(tx.Rebind(`
		SELECT id FROM destinations
		WHERE group_id = ? AND port = ? AND protocol = ?`),
		destination_group_id,
		port,
		protocol,
	).Scan(&id)
	return id, err
}
