package store

import (
	"fmt"
	"sort"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/policy-server/store/helpers"
)

//counterfeiter:generate -o fakes/security_groups_store.go --fake-name SecurityGroupsStore . SecurityGroupsStore
type SecurityGroupsStore interface {
	Replace([]SecurityGroup) error
	BySpaceGuids([]string, Page) ([]SecurityGroup, Pagination, error)
	LastUpdated() (int, error)
}

type SGStore struct {
	Conn Database
}

func (sgs *SGStore) BySpaceGuids(spaceGuids []string, page Page) ([]SecurityGroup, Pagination, error) {
	stagingSGs = 
	query := `
		SELECT
			security_groups.id,
			guid,
			name,
			rules,
			staging_default,
			running_default,
			staging_spaces.space_guid,
			running_spaces.space_guid
		FROM security_groups
			JOIN staging_spaces on staging_spaces.security_group_guid = security_groups.guid
		`

	whereClause := `staging_default=true OR running_default=true`

	if len(spaceGuids) > 0 {
		whereClause = fmt.Sprintf("%s OR staging_spaces.space_guid IN (%s) OR running_spaces.space_guid IN (%s)",
			whereClause,
			helpers.QuestionMarks(len(spaceGuids)),
			helpers.QuestionMarks(len(spaceGuids)),
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
		query = query + " AND security_groups.id >= %"
		whereBindings = append(whereBindings, page.From)
	}
	query = query + " ORDER BY security_groups.id"

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

	securityGroups := map[string]SecurityGroup{}
	stagingSpaces := map[string]map[string]struct{}{}
	runningSpaces := map[string]map[string]struct{}{}
	result := []SecurityGroup{}
	nextId := 0
	for rows.Next() {
		var id int
		var securityGroup SecurityGroup
		var stagingSpace, runningSpace *string
		err := rows.Scan(&id,
			&securityGroup.Guid,
			&securityGroup.Name,
			&securityGroup.Rules,
			&securityGroup.StagingDefault,
			&securityGroup.RunningDefault,
			&stagingSpace,
			&runningSpace,
		)
		if err != nil {
			return nil, Pagination{}, fmt.Errorf("scanning security group result: %s", err)
		}

		sg, ok := securityGroups[securityGroup.Guid]
		if !ok {
			securityGroup.StagingSpaceGuids = SpaceGuids{}
			securityGroup.RunningSpaceGuids = SpaceGuids{}
			sg = securityGroup
		}

		if stagingSpace != nil {
			if stagingSpaces[sg.Guid] == nil {
				stagingSpaces[sg.Guid] = map[string]struct{}{}
			}
			stagingSpaces[sg.Guid][*stagingSpace] = struct{}{}
		}
		if runningSpace != nil {
			if runningSpaces[sg.Guid] == nil {
				runningSpaces[sg.Guid] = map[string]struct{}{}
			}
			runningSpaces[sg.Guid][*runningSpace] = struct{}{}
		}

		if page.Limit == 0 || len(securityGroups) < page.Limit {
			securityGroups[sg.Guid] = sg
		} else {
			nextId = id
		}
	}
	for _, sg := range securityGroups {
		for space := range stagingSpaces[sg.Guid] {
			sg.StagingSpaceGuids = append(sg.StagingSpaceGuids, space)
		}
		sort.Strings(sg.StagingSpaceGuids)
		for space := range runningSpaces[sg.Guid] {
			sg.RunningSpaceGuids = append(sg.RunningSpaceGuids, space)
		}
		sort.Strings(sg.RunningSpaceGuids)
		result = append(result, sg)
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
		(guid, name, rules, staging_default, running_default)
		VALUES(?, ?, ?, ?, ?) ` +
		sgs.onConflictUpdateSQL() +
		` name=?, rules=?, staging_default=?, running_default=?`)

	for _, group := range newSecurityGroups {
		delete(existingGuids, group.Guid)

		_, err := tx.Exec(upsertQuery,
			group.Guid,
			group.Name,
			group.Rules,
			group.StagingDefault,
			group.RunningDefault,
			group.Name,
			group.Rules,
			group.StagingDefault,
			group.RunningDefault,
		)
		if err != nil {
			return fmt.Errorf("saving security group %s (%s): %s", group.Guid, group.Name, err)
		}

		err = sgs.replaceSpaceBinding(tx, "staging_spaces", group.Guid, group.StagingSpaceGuids)
		if err != nil {
			return fmt.Errorf("updating security group %s (%s)'s staging space bindings: %s", group.Guid, group.Name, err)
		}

		err = sgs.replaceSpaceBinding(tx, "running_spaces", group.Guid, group.RunningSpaceGuids)
		if err != nil {
			return fmt.Errorf("updating security group %s (%s)'s staging space bindings: %s", group.Guid, group.Name, err)
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

	err = sgs.updateLastUpdated(tx)
	if err != nil {
		return fmt.Errorf("updating security_groups_info.last_updated: %s", err)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}
	return nil
}

func (sgs *SGStore) replaceSpaceBinding(tx db.Transaction, tableName string, securityGroup string, spaceGuids []string) error {
	existingGuids := map[string]bool{}

	existingGuidQuery := tx.Rebind(fmt.Sprintf("SELECT space_guid id FROM %s WHERE security_group_guid = ?", tableName))
	rows, err := tx.Queryx(existingGuidQuery, securityGroup)
	if err != nil {
		return fmt.Errorf("selecting spaces from %s: %s", tableName, err)
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

	replaceQuery := tx.Rebind(fmt.Sprintf(`INSERT INTO %s (security_group_guid, space_guid) VALUES(?, ?)`, tableName) +
		sgs.onConflictUpdateSQL() + ` security_group_guid=?, space_guid=?`)

	for _, space := range spaceGuids {
		delete(existingGuids, space)
		_, err := tx.Exec(replaceQuery, securityGroup, space, securityGroup, space)
		if err != nil {
			return err
		}
	}

	if len(existingGuids) > 0 {
		args := []interface{}{securityGroup}
		for guid := range existingGuids {
			args = append(args, guid)
		}
		_, err = tx.Exec(tx.Rebind(fmt.Sprintf("DELETE FROM %s WHERE security_group_guid = ? and space_guid IN (%s)",
			tableName, helpers.QuestionMarks(len(existingGuids)))), args...)
		if err != nil {
			return fmt.Errorf("deleting security group %s' space bindings in %s: %s", securityGroup, tableName, err)
		}
	}

	return nil
}

// func (sgs *SGStore) jsonOverlapsSQL(columnName string, filterValues []string) string {
// 	switch sgs.Conn.DriverName() {
// 	case helpers.MySQL:
// 		clauses := []string{}
// 		for range filterValues {
// 			clauses = append(clauses, fmt.Sprintf(`json_contains(%s, json_quote(?))`, columnName))
// 		}
// 		return strings.Join(clauses, " OR ")
// 	case helpers.Postgres:
// 		filterList := helpers.MarksWithSeparator(len(filterValues), "%", ", ")
// 		return fmt.Sprintf(`%s ?| array[%s]`, columnName, filterList)
// 	default:
// 		return ""
// 	}
// }

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

func (sgs *SGStore) LastUpdated() (int, error) {
	var timestamp time.Time
	err := sgs.Conn.QueryRow(`SELECT last_updated FROM security_groups_info LIMIT 1`).Scan(&timestamp)
	if err != nil {
		return 0, fmt.Errorf("getting policies: %s", err)
	}
	return int(timestamp.UnixNano()), err
}

func (sgs *SGStore) updateLastUpdated(tx db.Transaction) error {
	_, err := tx.Exec(`UPDATE security_groups_info SET last_updated=CURRENT_TIMESTAMP(6)`)
	return err
}
