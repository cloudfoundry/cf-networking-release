package store

import (
	"database/sql"
	"fmt"
	"strings"

	"code.cloudfoundry.org/policy-server/store/helpers"
)

//counterfeiter:generate -o fakes/security_groups_store.go --fake-name SecurityGroupsStore . SecurityGroupsStore
type SecurityGroupsStore interface {
	BySpaceGuids([]string, Page) ([]SecurityGroup, Pagination, error)
	Replace([]SecurityGroup) error
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

/*
The Replace function replaces the security_group table with new data via the
following algorithm:
 1. make a map of all guids and ids of the current security groups
 2. make a temporary table called security_group_tmp
 3. fill in security_group_tmp table with newSecurityGroups, plus matching
 ids from security_group table
 4. rename the tables
	4a. security_groups --> security_groups_old
	4b. security_groups_tmp --> security_groups
 5. Drop the security_group_old table

This algorithm ensures that ids are kept the same, which is required because
policy-agent uses ids for pagination.
*/
func (sgs *SGStore) Replace(newSecurityGroups []SecurityGroup) error {
	guidsToIds, err := sgs.securityGroupGuidsToIds()
	if err != nil {
		return fmt.Errorf("getting security groups: %s", err)
	}

	tx, err := sgs.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(sgs.createTmpTableQuery())
	if err != nil {
		return fmt.Errorf("creating security_groups_tmp table: %s", err)
	}

	if sgs.Conn.DriverName() == helpers.MySQL {
		var maxId sql.NullInt64
		rows := tx.QueryRow("SELECT MAX(id) FROM security_groups")
		err = rows.Scan(&maxId)
		if err != nil {
			return fmt.Errorf("getting max id: %s", err)
		}
		if maxId.Valid {
			// Rebind does not work with auto_increment, we can trust id value
			_, err = tx.Exec(fmt.Sprintf(`ALTER TABLE security_groups_tmp AUTO_INCREMENT = %d`, maxId.Int64+1))
			if err != nil {
				return fmt.Errorf("setting auto_increment: %s", err)
			}
		}
	}

	insertQuery := tx.Rebind(`
		INSERT INTO security_groups_tmp
		(guid, name, rules, staging_default, running_default, staging_spaces, running_spaces)
		VALUES(?, ?, ?, ?, ?, ?, ?)`)

	insertWithIdQuery := tx.Rebind(`
		INSERT INTO security_groups_tmp
		(id, guid, name, rules, staging_default, running_default, staging_spaces, running_spaces)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`)

	for _, group := range newSecurityGroups {
		id, found := guidsToIds[group.Guid]
		if found {
			_, err = tx.Exec(insertWithIdQuery,
				id,
				group.Guid,
				group.Name,
				group.Rules,
				group.StagingDefault,
				group.RunningDefault,
				group.StagingSpaceGuids,
				group.RunningSpaceGuids,
			)

		} else {
			_, err = tx.Exec(insertQuery,
				group.Guid,
				group.Name,
				group.Rules,
				group.StagingDefault,
				group.RunningDefault,
				group.StagingSpaceGuids,
				group.RunningSpaceGuids,
			)
		}
		if err != nil {
			return fmt.Errorf("saving security group %s (%s): %s", group.Guid, group.Name, err)
		}
	}

	for _, query := range sgs.swapTablesQueries("security_groups", "security_groups_tmp", "security_groups_old") {
		_, err = tx.Exec(query)
		if err != nil {
			return fmt.Errorf("swapping security_groups_tmp and security_groups: %s", err)
		}
	}

	if sgs.Conn.DriverName() == helpers.Postgres {
		_, err = tx.Exec("ALTER SEQUENCE security_groups_id_seq OWNED BY security_groups.id")
		if err != nil {
			return fmt.Errorf("changing seq owner: %s", err)
		}
	}

	_, err = tx.Exec("DROP TABLE security_groups_old")
	if err != nil {
		return fmt.Errorf("dropping security_groups_old table: %s", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}
	return nil
}

func (sgs *SGStore) securityGroupGuidsToIds() (map[string]int, error) {
	rows, err := sgs.Conn.Query(`SELECT id, guid FROM security_groups`)
	if err != nil {
		return nil, fmt.Errorf("selecting security groups: %s", err)
	}
	defer rows.Close()

	result := map[string]int{}
	var id int
	var guid string

	for rows.Next() {
		err := rows.Scan(&id, &guid)
		if err != nil {
			return nil, fmt.Errorf("scanning security group result: %s", err)
		}
		result[guid] = id
	}

	return result, nil
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
		filterList := fmt.Sprintf("%s", helpers.MarksWithSeparator(len(filterValues), "%", ", "))
		return fmt.Sprintf(`%s ?| array[%s]`, columnName, filterList)
	default:
		return ""
	}
}

func (sgs *SGStore) swapTablesQueries(currentTableName, tempTableName, oldTableName string) []string {
	switch sgs.Conn.DriverName() {
	case helpers.MySQL:
		return []string{
			"RENAME TABLE " + currentTableName + " TO " + oldTableName + ", " + tempTableName + " TO " + currentTableName,
		}
	case helpers.Postgres:
		return []string{
			"ALTER TABLE " + currentTableName + " RENAME TO " + oldTableName,
			"ALTER TABLE " + tempTableName + " RENAME TO " + currentTableName,
		}
	default:
		return []string{""}
	}
}

func (sgs *SGStore) createTmpTableQuery() string {
	switch sgs.Conn.DriverName() {
	case helpers.MySQL:
		return "CREATE TABLE security_groups_tmp LIKE security_groups"
	case helpers.Postgres:
		return "CREATE TABLE security_groups_tmp (LIKE security_groups INCLUDING ALL)"
	default:
		return ""
	}
}
