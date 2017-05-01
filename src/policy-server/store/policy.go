package store

import "context"

//go:generate counterfeiter -o fakes/policy_repo.go --fake-name PolicyRepo . PolicyRepo
type PolicyRepo interface {
	Create(context.Context, Transaction, int, int) error
	Delete(context.Context, Transaction, int, int) error
	CountWhereGroupID(context.Context, Transaction, int) (int, error)
	CountWhereDestinationID(context.Context, Transaction, int) (int, error)
}

type Policy struct {
}

func (p *Policy) Create(ctx context.Context, tx Transaction, source_group_id int, destination_id int) error {
	_, err := tx.ExecContext(ctx, tx.Rebind(`
		INSERT INTO policies (group_id, destination_id)
		SELECT ?, ?
		WHERE
		NOT EXISTS (
			SELECT *
			FROM policies
			WHERE group_id = ? AND destination_id = ?
		)`),
		source_group_id,
		destination_id,
		source_group_id,
		destination_id,
	)
	return err
}

func (p *Policy) Delete(ctx context.Context, tx Transaction, source_group_id int, destination_id int) error {
	_, err := tx.ExecContext(ctx, tx.Rebind(`DELETE FROM policies WHERE group_id = ? AND destination_id = ?`),
		source_group_id,
		destination_id,
	)
	return err
}

func (p *Policy) CountWhereGroupID(ctx context.Context, tx Transaction, source_group_id int) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx,
		tx.Rebind(`SELECT COUNT(*) FROM policies WHERE group_id = ?`),
		source_group_id,
	).Scan(&count)

	return count, err
}

func (p *Policy) CountWhereDestinationID(ctx context.Context, tx Transaction, destination_id int) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx,
		tx.Rebind(`SELECT COUNT(*) FROM policies WHERE destination_id = ?`),
		destination_id,
	).Scan(&count)

	return count, err
}
