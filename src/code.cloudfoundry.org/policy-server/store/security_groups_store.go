package store

import (
	"fmt"
	"strings"

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
	query := `
		SELECT
			id,
			guid,
			name,
			rules,
			staging_default,
			running_default,
			staging_spaces,
			running_spaces
		FROM security_groups`

	whereClause := `staging_default=true OR running_default=true`

	if len(spaceGuids) > 0 {
		whereClause = fmt.Sprintf("%s OR %s OR %s",
			whereClause,
			sgs.jsonOverlapsSQL("staging_spaces", spaceGuids),
			sgs.jsonOverlapsSQL("running_spaces", spaceGuids),
		)
	}

	query = fmt.Sprintf("%s WHERE (%s)", query, whereClause)

	// one for running and one for staging
	whereBindings := make([]interface{}, len(spaceGuids)*2)
	for i, spaceGuid := range spaceGuids {
		whereBindings[i] = spaceGuid
		whereBindings[i+len(spaceGuids)] = spaceGuid
	}

	if page.From > 0 {
		query = query + " AND id >= %"
		whereBindings = append(whereBindings, page.From)
	}
	query = query + " ORDER BY id"

	if page.Limit > 0 {
		// we don't use a placeholder because limit is an integer and it is safe to interpolate it
		query = fmt.Sprintf(`%s LIMIT %d`, query, page.Limit+1)
	}

	rebindedQuery := helpers.RebindForSQLDialectAndMark(query, sgs.Conn.DriverName(), "%")

	rows, err := sgs.Conn.Query(rebindedQuery, whereBindings...)
	if err != nil {
		return nil, Pagination{}, fmt.Errorf("selecting security groups: %s", err)
	}
	defer rows.Close()

	result := []SecurityGroup{}
	nextId := 0
	for rows.Next() {
		var id int
		var securityGroup SecurityGroup
		err := rows.Scan(&id,
			&securityGroup.Guid,
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

		if page.Limit == 0 || len(result) < page.Limit {
			result = append(result, securityGroup)
		} else {
			nextId = id
		}
	}
	return result, Pagination{Next: nextId}, nil
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

	upsertQuery := tx.Rebind(`
		INSERT INTO security_groups
		(guid, name, rules, staging_default, running_default, staging_spaces, running_spaces)
		VALUES(?, ?, ?, ?, ?, ?, ?) ` +
		sgs.onConflictUpdateSQL() +
		` name=?, rules=?, staging_default=?, running_default=?, staging_spaces=?, running_spaces=?`)

	for _, group := range newSecurityGroups {
		delete(existingGuids, group.Guid)

		_, err := tx.Exec(upsertQuery,
			group.Guid,
			group.Name,
			group.Rules,
			group.StagingDefault,
			group.RunningDefault,
			group.StagingSpaceGuids,
			group.RunningSpaceGuids,
			group.Name,
			group.Rules,
			group.StagingDefault,
			group.RunningDefault,
			group.StagingSpaceGuids,
			group.RunningSpaceGuids,
		)
		if err != nil {
			return fmt.Errorf("saving security group %s (%s): %s", group.Guid, group.Name, err)
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

func (sgs *SGStore) jsonOverlapsSQL(columnName string, filterValues []string) string {
	switch sgs.Conn.DriverName() {
	case helpers.MySQL:
		clauses := []string{}
		for range filterValues {
			clauses = append(clauses, fmt.Sprintf(`json_contains(%s, json_quote(?))`, columnName))
		}
		return strings.Join(clauses, " OR ")
	case helpers.Postgres:
		filterList := helpers.MarksWithSeparator(len(filterValues), "%", ", ")
		return fmt.Sprintf(`%s ?| array[%s]`, columnName, filterList)
	default:
		return ""
	}
}

func (sgs *SGStore) onConflictUpdateSQL() string {
	switch sgs.Conn.DriverName() {
	case helpers.MySQL:
		return "ON DUPLICATE KEY UPDATE"
	case helpers.Postgres:
		return "ON CONFLICT (guid) DO UPDATE SET"
	default:
		return ""
	}
}
