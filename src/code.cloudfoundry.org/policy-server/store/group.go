package store

import (
	"database/sql"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

const mysqlErrorCode = "1062"
const postgresErrorCode = "23505"

//counterfeiter:generate -o fakes/group_repo.go --fake-name GroupRepo . GroupRepo
type GroupRepo interface {
	Create(db.Transaction, string, string) (int, error)
	Delete(db.Transaction, int) error
	GetID(db.Transaction, string) (int, error)
}

type GroupTable struct {
}

func (g *GroupTable) Create(tx db.Transaction, guid, groupType string) (int, error) {
	id, err := g.findIDByGuidAndType(tx, guid, groupType)
	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return -1, err
	}

	id, err = g.firstBlankRow(tx)
	if err != nil {
		return -1, fmt.Errorf("failed to find available tag: %s", err.Error())
	}

	err = g.updateRow(tx, id, guid, groupType)
	if err == nil {
		return id, nil
	}

	id, returnedGroupType, _ := g.GetIDAndGroupType(tx, guid)
	if isDuplicateError(err) && returnedGroupType == groupType {
		return id, nil
	}

	return -1, err
}

func isDuplicateError(err error) bool {
	return strings.Contains(err.Error(), mysqlErrorCode) || strings.Contains(err.Error(), postgresErrorCode)
}

func (g *GroupTable) findIDByGuidAndType(tx db.Transaction, guid, groupType string) (int, error) {
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

func (g *GroupTable) GetIDAndGroupType(tx db.Transaction, guid string) (int, string, error) {
	var id int
	var groupType string
	err := tx.QueryRow(
		tx.Rebind(`SELECT id, type FROM "groups" WHERE guid = ?`),
		guid,
	).Scan(&id, &groupType)

	return id, groupType, err
}
