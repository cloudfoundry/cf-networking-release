package store

import "fmt"

type SecurityGroupsStore struct {
	Conn Database
}

func (sgs *SecurityGroupsStore) Replace(newSecurityGroups []SecurityGroup) error {
	tx, err := sgs.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}
	defer tx.Rollback()

	// delete all existing SGs and space associations
	_, err = tx.Exec(tx.Rebind("DELETE from running_security_groups_spaces"))
	if err != nil {
		return fmt.Errorf("deleting running security group associations: %s", err)
	}
	_, err = tx.Exec(tx.Rebind("DELETE from staging_security_groups_spaces"))
	if err != nil {
		return fmt.Errorf("deleting staging security group associations: %s", err)
	}
	_, err = tx.Exec(tx.Rebind("DELETE from security_groups"))
	if err != nil {
		return fmt.Errorf("deleting security groups: %s", err)
	}

	// loop groups
	for _, group := range newSecurityGroups {
		_, err := tx.Exec(tx.Rebind(`
			INSERT INTO security_groups
			(guid, name, rules, running_default, staging_default)
			VALUES(?, ?, ?, ?, ?)`),
			group.Guid,
			group.Name,
			group.Rules,
			group.RunningDefault,
			group.StagingDefault,
		)
		if err != nil {
			return fmt.Errorf("adding new security group %s (%s): %s", group.Guid, group.Name, err)
		}

		for _, spaceGuid := range group.StagingSpaceGuids {
			_, err := tx.Exec(tx.Rebind(`
				INSERT INTO staging_security_groups_spaces
				(space_guid, security_group_guid) VALUES(?, ?)`),
				spaceGuid,
				group.Guid,
			)
			if err != nil {
				return fmt.Errorf("adding staging security group association for %s (%s) to space %s: %s",
					group.Guid,
					group.Name,
					spaceGuid,
					err,
				)
			}
		}
		for _, spaceGuid := range group.RunningSpaceGuids {
			_, err := tx.Exec(tx.Rebind(`
				INSERT INTO running_security_groups_spaces
				(space_guid, security_group_guid) VALUES(?, ?)`),
				spaceGuid,
				group.Guid,
			)
			if err != nil {
				return fmt.Errorf("adding running security group association for %s (%s) to space %s: %s",
					group.Guid,
					group.Name,
					spaceGuid,
					err,
				)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}
	return nil
}
