package store

import (
	"database/sql"
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

//counterfeiter:generate -o fakes/group_repo.go --fake-name GroupRepo . GroupRepo
type GroupRepo interface {
	Create(db.Transaction, string, string) (int, error)
	Delete(db.Transaction, int) error
	GetID(db.Transaction, string) (int, error)
}

type GroupTable struct {
}

func (g *GroupTable) Create(tx db.Transaction, guid, groupType string) (int, error) {
	id, err := g.findRowByGUID(tx, guid, groupType)
	if err != nil {
		if err == sql.ErrNoRows {
			id, err = g.firstBlankRow(tx)
			if err != nil {
				return -1, fmt.Errorf("failed to find available tag: %s", err.Error())
			} else {
				err = g.updateRow(tx, id, guid, groupType)
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

func (g *GroupTable) findRowByGUID(tx db.Transaction, guid, groupType string) (int, error) {
	var id int
	err := tx.QueryRow(
		tx.Rebind(`
		SELECT id FROM "groups"
		WHERE guid = ? AND type = ?
		`),
		guid,
		groupType,
	).Scan(&id)
	return id, err
}

func (g *GroupTable) firstBlankRow(tx db.Transaction) (int, error) {
	var id int
	err := tx.QueryRow(
		`SELECT id FROM "groups"
		WHERE guid is NULL
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`).Scan(&id)
	return id, err
}

func (g *GroupTable) updateRow(tx db.Transaction, id int, guid, groupType string) error {
	_, err := tx.Exec(
		tx.Rebind(`
			UPDATE "groups" SET guid = ?, type =  ?
			WHERE id = ?
		`),
		guid,
		groupType,
		id,
	)
	return err
}

func (g *GroupTable) Delete(tx db.Transaction, id int) error {
	_, err := tx.Exec(
		tx.Rebind(`UPDATE "groups" SET guid = NULL, type = NULL WHERE id = ?`),
		id,
	)
	return err
}

func (g *GroupTable) GetID(tx db.Transaction, guid string) (int, error) {
	var id int
	err := tx.QueryRow(
		tx.Rebind(`SELECT id FROM "groups" WHERE guid = ?`),
		guid,
	).Scan(&id)

	return id, err
}
