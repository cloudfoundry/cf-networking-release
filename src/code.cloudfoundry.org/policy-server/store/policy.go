package store

import "code.cloudfoundry.org/cf-networking-helpers/db"

//counterfeiter:generate -o fakes/policy_repo.go --fake-name PolicyRepo . PolicyRepo
type PolicyRepo interface {
	Create(db.Transaction, int, int) error
	Delete(db.Transaction, int, int) error
	CountWhereGroupID(db.Transaction, int) (int, error)
	CountWhereDestinationID(db.Transaction, int) (int, error)
}

type PolicyTable struct {
}

func (p *PolicyTable) Create(tx db.Transaction, sourceGroupId int, destinationId int) error {
	dualStatement := ""
	if tx.DriverName() == "mysql" {
		dualStatement = " FROM DUAL "
	}

	_, err := tx.Exec(tx.Rebind(`
		INSERT INTO policies (group_id, destination_id)
		SELECT ?, ? `+dualStatement+`
		WHERE
		NOT EXISTS (
			SELECT *
			FROM policies
			WHERE group_id = ? AND destination_id = ?
		)`),
		sourceGroupId,
		destinationId,
		sourceGroupId,
		destinationId,
	)
	return err
}

func (p *PolicyTable) Delete(tx db.Transaction, sourceGroupId int, destinationId int) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM policies WHERE group_id = ? AND destination_id = ?`),
		sourceGroupId,
		destinationId,
	)
	return err
}

func (p *PolicyTable) CountWhereGroupID(tx db.Transaction, sourceGroupId int) (int, error) {
	var count int
	err := tx.QueryRow(
		tx.Rebind(`SELECT COUNT(*) FROM policies WHERE group_id = ?`),
		sourceGroupId,
	).Scan(&count)

	return count, err
}

func (p *PolicyTable) CountWhereDestinationID(tx db.Transaction, destinationId int) (int, error) {
	var count int
	err := tx.QueryRow(
		tx.Rebind(`SELECT COUNT(*) FROM policies WHERE destination_id = ?`),
		destinationId,
	).Scan(&count)

	return count, err
}
