package store

//go:generate counterfeiter -o ../fakes/destination_creator.go --fake-name DestinationCreator . DestinationCreator
type DestinationCreator interface {
	Create(Transaction, int, int, string) (int, error)
}

type Destination struct {
}

func (d *Destination) Create(tx Transaction, destination_group_id int, port int, protocol string) (int, error) {
	_, err := tx.Exec(
		`INSERT INTO destinations (group_id, port, protocol)
				SELECT $1, $2, $3
				WHERE
					NOT EXISTS (
						SELECT *
						FROM destinations
						WHERE group_id = $1 AND port = $2 AND protocol = $3
					)`,
		destination_group_id,
		port,
		protocol,
	)
	if err != nil {
		return -1, err
	}

	var id int
	err = tx.QueryRow(
		`SELECT id FROM destinations
				WHERE group_id = $1 AND port = $2 AND protocol = $3`,
		destination_group_id,
		port,
		protocol,
	).Scan(&id)

	return id, err
}
