package store

import (
	"fmt"

	"code.cloudfoundry.org/policy-server/store/helpers"
)

//counterfeiter:generate -o fakes/security_groups_store.go --fake-name SecurityGroupsStore . SecurityGroupsStore
type SecurityGroupsStore interface {
	Replace([]SecurityGroup) error
	BySpaceGuids([]string, Page) ([]SecurityGroup, Pagination, error)
}

type SGStore struct {
	Conn Database
}

func (sgs *SGStore) BySpaceGuids(spaceGuids []string, page Page) ([]SecurityGroup, Pagination, error) {
	if len(spaceGuids) == 0 {
		return nil, Pagination{}, nil
	}

	whereClause := fmt.Sprintf("array[%s]", helpers.MarksWithSeparator(len(spaceGuids), "%", ", "))

	query := `
		SELECT
			guid,
			name,
			rules,
			staging_default,
			running_default,
			staging_spaces,
			running_spaces
		FROM security_groups
		WHERE staging_spaces ?| ` + whereClause + ` OR running_spaces ?| ` + whereClause

	// one for running and one for staging
	whereBindings := make([]interface{}, len(spaceGuids)*2)
	for i, spaceGuid := range spaceGuids {
		whereBindings[i] = spaceGuid
		whereBindings[i+len(spaceGuids)] = spaceGuid
	}

	rebindedQuery := helpers.RebindForSQLDialectTwo(query, sgs.Conn.DriverName())
	rows, err := sgs.Conn.Query(rebindedQuery, whereBindings...)
	if err != nil {
		return nil, Pagination{}, fmt.Errorf("selecting security groups: %s", err)
	}
	defer rows.Close()

	result := []SecurityGroup{}
	for rows.Next() {
		var securityGroup SecurityGroup
		err := rows.Scan(&securityGroup.Guid,
			&securityGroup.Name,
			&securityGroup.Rules,
			&securityGroup.StagingDefault,
			&securityGroup.RunningDefault,
			&securityGroup.StagingSpaceGuids,
			&securityGroup.RunningSpaceGuids,
		)
		if err != nil {
			return nil, Pagination{}, fmt.Errorf("scanning security group result: %s", err)
		}

		result = append(result, securityGroup)
	}

	return result, Pagination{}, nil
}

func (sgs *SGStore) Replace(newSecurityGroups []SecurityGroup) error {
	tx, err := sgs.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}
	defer tx.Rollback()

	existingGuids := map[string]bool{}
	rows, err := tx.Queryx("SELECT guid FROM security_groups")
	if err != nil {
		return fmt.Errorf("selecting security groups: %s", err)
	}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var guid string
			err := rows.Scan(&guid)
			if err != nil {
				return fmt.Errorf("scanning security group result: %s", err)
			}
			existingGuids[guid] = true
		}
	}

	updateQuery := tx.Rebind(`
		UPDATE security_groups SET name=?, rules=?, staging_default=?, running_default=?, staging_spaces=?, running_spaces=?
		WHERE guid=?`)
	insertQuery := tx.Rebind(`
		INSERT INTO security_groups
		(guid, name, rules, staging_default, running_default, staging_spaces, running_spaces)
		VALUES(?, ?, ?, ?, ?, ?, ?)`)

	for _, group := range newSecurityGroups {
		delete(existingGuids, group.Guid)

		result, err := tx.Exec(updateQuery,
			group.Name,
			group.Rules,
			group.StagingDefault,
			group.RunningDefault,
			group.StagingSpaceGuids,
			group.RunningSpaceGuids,
			group.Guid,
		)
		if err != nil {
			return fmt.Errorf("updating security group %s (%s): %s", group.Guid, group.Name, err)
		}
		var affectedRows int64
		if result != nil {
			affectedRows, _ = result.RowsAffected()
		}
		if affectedRows == 0 {
			_, err = tx.Exec(insertQuery,
				group.Guid,
				group.Name,
				group.Rules,
				group.StagingDefault,
				group.RunningDefault,
				group.StagingSpaceGuids,
				group.RunningSpaceGuids,
			)
			if err != nil {
				return fmt.Errorf("adding new security group %s (%s): %s", group.Guid, group.Name, err)
			}
		}
	}

	if len(existingGuids) > 0 {
		guids := []interface{}{}
		for guid := range existingGuids {
			guids = append(guids, guid)
		}
		_, err = tx.Exec(tx.Rebind(`
			DELETE FROM security_groups WHERE guid IN (`+helpers.QuestionMarks(len(existingGuids))+`)`),
			guids...)
		if err != nil {
			return fmt.Errorf("deleting security groups: %s", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}
	return nil
}
