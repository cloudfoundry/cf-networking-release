package store

//go:generate counterfeiter -o ../fakes/group_creator.go --fake-name GroupCreator . GroupCreator
type GroupCreator interface {
	Create(Transaction, string) (int, error)
}

type Group struct {
}

func (g *Group) Create(tx Transaction, guid string) (int, error) {
	_, err := tx.Exec(tx.Rebind(`
		INSERT INTO groups (guid) SELECT ?
		WHERE
		NOT EXISTS (
			SELECT guid FROM groups WHERE guid = ?
		)`),
		guid,
		guid,
	)
	if err != nil {
		return -1, err
	}

	var id int
	err = tx.QueryRow(
		tx.Rebind(`SELECT id FROM groups WHERE guid = ?`),
		guid,
	).Scan(&id)

	return id, err
}
