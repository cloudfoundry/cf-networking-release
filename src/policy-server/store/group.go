package store

import (
	"database/sql"
	"errors"
)

//go:generate counterfeiter -o ../fakes/group_repo.go --fake-name GroupRepo . GroupRepo
type GroupRepo interface {
	Create(Transaction, string) (int, error)
	Delete(Transaction, int) error
	GetID(Transaction, string) (int, error)
}

type Group struct {
}

func (g *Group) Create(tx Transaction, guid string) (int, error) {
	id, err := g.findRowByGUID(tx, guid)
	if err != nil {
		if err == sql.ErrNoRows {
			id, err = g.firstBlankRow(tx)
			if err != nil {
				return -1, errors.New("failed to find available tag")
			} else {
				err = g.updateRow(tx, id, guid)
				if err != nil {
					return -1, err
				}
				return id, nil
			}
		}
		return -1, err
	}
	return id, nil
}

func (g *Group) findRowByGUID(tx Transaction, guid string) (int, error) {
	var id int
	err := tx.QueryRow(
		tx.Rebind(`
		SELECT id FROM groups
		WHERE guid = ?
		`),
		guid,
	).Scan(&id)
	return id, err
}

func (g *Group) firstBlankRow(tx Transaction) (int, error) {
	var id int
	err := tx.QueryRow(`
		SELECT id FROM groups
		WHERE guid is NULL
		ORDER BY id
		LIMIT 1
	`).Scan(&id)
	return id, err
}

func (g *Group) updateRow(tx Transaction, id int, guid string) error {
	_, err := tx.Exec(
		tx.Rebind(`
			UPDATE groups SET guid = ?
			WHERE id = ?
		`),
		guid,
		id,
	)
	return err
}

func (g *Group) Delete(tx Transaction, id int) error {
	_, err := tx.Exec(
		tx.Rebind(`UPDATE groups SET guid = NULL WHERE id = ?`),
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
