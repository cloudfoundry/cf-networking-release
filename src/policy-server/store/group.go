package store

//go:generate counterfeiter -o ../fakes/group_repo.go --fake-name GroupRepo . GroupRepo
type GroupRepo interface {
	Create(Transaction, string) (int, error)
	Delete(Transaction, int) error
	GetID(Transaction, string) (int, error)
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

	return g.GetID(tx, guid)
}

func (g *Group) Delete(tx Transaction, id int) error {
	_, err := tx.Exec(
		tx.Rebind(`DELETE FROM groups WHERE id = ?`),
		id,
	)
	return err
}

func (g *Group) GetID(tx Transaction, guid string) (int, error) {
	var id int
	err := tx.QueryRow(
		tx.Rebind(`SELECT id FROM groups WHERE guid = ?`),
		guid,
	).Scan(&id)

	return id, err
}
