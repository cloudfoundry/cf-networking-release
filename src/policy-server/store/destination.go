package store

import "context"

//go:generate counterfeiter -o fakes/destination_repo.go --fake-name DestinationRepo . DestinationRepo
type DestinationRepo interface {
	Create(context.Context, Transaction, int, int, string) (int, error)
	Delete(context.Context, Transaction, int) error
	GetID(context.Context, Transaction, int, int, string) (int, error)
	CountWhereGroupID(context.Context, Transaction, int) (int, error)
}

type Destination struct {
}

func (d *Destination) Create(ctx context.Context, tx Transaction, destination_group_id int, port int, protocol string) (int, error) {
	_, err := tx.ExecContext(ctx, tx.Rebind(`
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
	id, err := d.GetID(ctx, tx, destination_group_id, port, protocol)
	return id, err
}

func (d *Destination) Delete(ctx context.Context, tx Transaction, id int) error {
	_, err := tx.ExecContext(ctx,
		tx.Rebind(`DELETE FROM destinations WHERE id = ?`),
		id,
	)
	return err
}

func (d *Destination) GetID(ctx context.Context, tx Transaction, destination_group_id int, port int, protocol string) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, tx.Rebind(`
		SELECT id FROM destinations
		WHERE group_id = ? AND port = ? AND protocol = ? FOR UPDATE`),
		destination_group_id,
		port,
		protocol,
	).Scan(&id)
	return id, err
}

func (d *Destination) CountWhereGroupID(ctx context.Context, tx Transaction, group_id int) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx,
		tx.Rebind(`SELECT COUNT(*) FROM destinations WHERE group_id = ?`),
		group_id,
	).Scan(&count)
	return count, err
}
