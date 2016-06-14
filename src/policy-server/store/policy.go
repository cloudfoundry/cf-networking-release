package store

//go:generate counterfeiter -o ../fakes/policy_creator.go --fake-name PolicyCreator . PolicyCreator
type PolicyCreator interface {
	Create(Transaction, int, int) error
}

type Policy struct {
}

func (p *Policy) Create(tx Transaction, source_group_id int, destination_id int) error {
	_, err := tx.Exec(
		`INSERT INTO policies (group_id, destination_id)
				SELECT $1, $2
				WHERE
					NOT EXISTS (
						SELECT *
						FROM policies
						WHERE group_id = $1 AND destination_id = $2
					)`,
		source_group_id,
		destination_id,
	)
	if err != nil {
		return err
	}

	return nil
}
