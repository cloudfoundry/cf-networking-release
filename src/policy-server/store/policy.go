package store

//go:generate counterfeiter -o ../fakes/policy_repo.go --fake-name PolicyRepo . PolicyRepo
type PolicyRepo interface {
	Create(Transaction, int, int) error
	Delete(Transaction, int, int) error
}

type Policy struct {
}

func (p *Policy) Create(tx Transaction, source_group_id int, destination_id int) error {
	_, err := tx.Exec(tx.Rebind(`
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
	if err != nil {
		return err
	}

	return nil
}

func (p *Policy) Delete(tx Transaction, source_group_id int, destination_id int) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM policies WHERE group_id = ? AND destination_id = ?`),
		source_group_id,
		destination_id,
	)
	return err
}
