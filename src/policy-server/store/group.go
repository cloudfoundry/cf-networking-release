package store

import (
	"context"
	"database/sql"
	"fmt"
)

//go:generate counterfeiter -o fakes/group_repo.go --fake-name GroupRepo . GroupRepo
type GroupRepo interface {
	Create(context.Context, Transaction, string) (int, error)
	Delete(context.Context, Transaction, int) error
	GetID(context.Context, Transaction, string) (int, error)
}

type Group struct {
}

func (g *Group) Create(ctx context.Context, tx Transaction, guid string) (int, error) {
	id, err := g.findRowByGUID(ctx, tx, guid)
	if err != nil {
		if err == sql.ErrNoRows {
			id, err = g.firstBlankRow(ctx, tx)
			if err != nil {
				return -1, fmt.Errorf("failed to find available tag: %s", err.Error())
			} else {
				err = g.updateRow(ctx, tx, id, guid)
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

func (g *Group) findRowByGUID(ctx context.Context, tx Transaction, guid string) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx,
		tx.Rebind(`
		SELECT id FROM groups
		WHERE guid = ?
		`),
		guid,
	).Scan(&id)
	return id, err
}

func (g *Group) firstBlankRow(ctx context.Context, tx Transaction) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM groups
		WHERE guid is NULL
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`).Scan(&id)
	return id, err
}

func (g *Group) updateRow(ctx context.Context, tx Transaction, id int, guid string) error {
	_, err := tx.ExecContext(ctx,
		tx.Rebind(`
			UPDATE groups SET guid = ?
			WHERE id = ?
		`),
		guid,
		id,
	)
	return err
}

func (g *Group) Delete(ctx context.Context, tx Transaction, id int) error {
	_, err := tx.ExecContext(ctx,
		tx.Rebind(`UPDATE groups SET guid = NULL WHERE id = ?`),
		id,
	)
	return err
}

func (g *Group) GetID(ctx context.Context, tx Transaction, guid string) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx,
		tx.Rebind(`SELECT id FROM groups WHERE guid = ?`),
		guid,
	).Scan(&id)

	return id, err
}
