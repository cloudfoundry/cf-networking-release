package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/store/helpers"
)

//counterfeiter:generate -o fakes/security_groups_store.go --fake-name SecurityGroupsStore . SecurityGroupsStore
type SecurityGroupsStore interface {
	Replace([]SecurityGroup) error
	BySpaceGuids([]string, Page) ([]SecurityGroup, Pagination, error)
	LastUpdated() (int, error)
}

type SGStore struct {
	Logger lager.Logger
	Conn   Database
}

func buildBoundASGQuery(table string, spaceGuids []string) string {
	return fmt.Sprintf("SELECT security_group_guid AS guid FROM %s_security_groups_spaces WHERE space_guid IN (%s)", table, helpers.QuestionMarks(len(spaceGuids)))
}

func (sgs *SGStore) BySpaceGuids(spaceGuids []string, page Page) ([]SecurityGroup, Pagination, error) {
	var boundASGQuery string
	if len(spaceGuids) > 0 {
		boundASGQuery = fmt.Sprintf(`
	UNION
		%s
	UNION
		%s`, buildBoundASGQuery("staging", spaceGuids), buildBoundASGQuery("running", spaceGuids))
	}
	query := fmt.Sprintf(`SELECT
 sgs.id,
 sgs.guid,
 sgs.name,
 sgs.rules,
 sgs.staging_default,
 sgs.running_default,
 sgs.staging_spaces,
 sgs.running_spaces
FROM security_groups AS sgs WHERE guid in (
	SELECT guid FROM (
		SELECT guid FROM security_groups WHERE staging_default = true
	UNION
		SELECT guid FROM security_groups WHERE running_default = true
%s
	) as bound
)
`, boundASGQuery)

	whereBindings := make([]any, len(spaceGuids))
	for i, spaceGuid := range spaceGuids {
		whereBindings[i] = spaceGuid
	}
	// add a second set for running space guids
	whereBindings = append(whereBindings, whereBindings...)

	if page.From > 0 {
		query = query + " AND sgs.id >= ?"
		whereBindings = append(whereBindings, page.From)
	}
	query = query + " ORDER BY sgs.id"

	if page.Limit > 0 {
		// we don't use a placeholder because limit is an integer and it is safe to interpolate it
		query = fmt.Sprintf(`%s LIMIT %d`, query, page.Limit+1)
	}

	rebindedQuery := helpers.RebindForSQLDialect(query, sgs.Conn.DriverName())

	rows, err := sgs.Conn.Query(rebindedQuery, whereBindings...)
	if err != nil {
		sgs.Logger.Error("selecting-security-groups", err, lager.Data{"query": rebindedQuery})
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

func calculateAsgHash(group SecurityGroup) (string, error) {
	slices.Sort(group.RunningSpaceGuids)
	slices.Sort(group.StagingSpaceGuids)
	groupJson, err := json.Marshal(group)
	if err != nil {
		return "", fmt.Errorf("failed-marshaling-asg-as-json: %s", err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(groupJson))
	return hash, nil
}

func (sgs *SGStore) Replace(newSecurityGroups []SecurityGroup) error {
	tx, err := sgs.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}
	defer tx.Rollback()

	if len(newSecurityGroups) == 0 {
		_, err = tx.Exec("DELETE FROM security_groups")
		if err != nil {
			return fmt.Errorf("deleting ALL security groups: %s", err)
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("committing transaction to delete ALL security groups: %s", err)
		}
		return nil
	}

	existingGuids := map[string]string{}
	rows, err := tx.Queryx("SELECT guid, hash FROM security_groups")
	if err != nil {
		return fmt.Errorf("selecting security groups: %s", err)
	}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var guid, hash string
			err := rows.Scan(&guid, &hash)
			if err != nil {
				return fmt.Errorf("scanning security group result: %s", err)
			}
			existingGuids[guid] = hash
		}
	}

	upsertQuery := `
		INSERT INTO security_groups
		(guid, name, hash, rules, staging_default, running_default, staging_spaces, running_spaces)
		VALUES`
	columnsPerRecord := 8
	onConflictQuery := sgs.onConflictUpdateSQL("name", "hash", "rules", "staging_default", "running_default", "staging_spaces", "running_spaces")

	stagingBindings := map[string][]string{}
	runningBindings := map[string][]string{}
	var insertValues []any
	for _, group := range newSecurityGroups {
		originalHash := existingGuids[group.Guid]
		delete(existingGuids, group.Guid)

		newHash, err := calculateAsgHash(group)
		if err != nil {
			return fmt.Errorf("failed-calculating-asg-hash: %s", err)
		}
		if newHash != originalHash {
			insertValues = append(insertValues,
				group.Guid,
				group.Name,
				newHash,
				group.Rules,
				group.StagingDefault,
				group.RunningDefault,
				group.StagingSpaceGuids,
				group.RunningSpaceGuids,
			)

			stagingBindings[group.Guid] = group.StagingSpaceGuids
			runningBindings[group.Guid] = group.RunningSpaceGuids
		}
	}

	if len(insertValues) > 0 {
		sgs.Logger.Debug("updating-existing-security-groups", lager.Data{"": len(insertValues) / columnsPerRecord})
		err = sgs.BatchPreparedStatement(tx, upsertQuery, onConflictQuery, insertValues, columnsPerRecord)
		if err != nil {
			return fmt.Errorf("upserting security groups: %s", err)
		}

		err = sgs.ReplaceSecurityGroupSpaceAssociations(tx, "staging_security_groups_spaces", stagingBindings)
		if err != nil {
			return fmt.Errorf("replacing staging space associations: %s", err)
		}
		err = sgs.ReplaceSecurityGroupSpaceAssociations(tx, "running_security_groups_spaces", runningBindings)
		if err != nil {
			return fmt.Errorf("replacing running space associations: %s", err)
		}
	}

	if len(existingGuids) > 0 {
		sgs.Logger.Debug("deleting-stale-security-groups", lager.Data{"num_records": len(insertValues) / columnsPerRecord})
		guidsToDelete := []any{}
		for guid := range existingGuids {
			guidsToDelete = append(guidsToDelete, guid)
		}

		err = sgs.BatchPreparedStatement(tx, "DELETE FROM security_groups WHERE guid IN (", ")", guidsToDelete, 1)
		if err != nil {
			return fmt.Errorf("deleting security groups: %s", err)
		}
	}

	err = sgs.updateLastUpdated(tx)
	if err != nil {
		return fmt.Errorf("updating security_groups_info.last_updated: %s", err)
	}
	sgs.Logger.Debug("committing-transaction")
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}
	return nil
}

func (sgs *SGStore) BatchPreparedStatement(tx db.Transaction, statementStart, statementEnd string, parameterValues []any, parametersPerRecord int) error {
	if len(parameterValues) == 0 {
		return nil
	}
	// Both mysql + postgres claim to use 16bit integers in the protocol spec to identify how many
	// parameters are being provided, 0 indicating no parameters, and a max of 65535.
	parameterLimit := 65535

	maxRecordCount := parameterLimit / parametersPerRecord
	parametersPerBatch := maxRecordCount * parametersPerRecord

	batchNumber := 1
	for i := 0; i < len(parameterValues); i += parametersPerBatch {
		lastIndex := min(i+parametersPerBatch, len(parameterValues))
		recordCount := min((lastIndex-i)/parametersPerRecord, maxRecordCount)
		values := parameterValues[i:lastIndex]

		reboundStatement := tx.Rebind(
			fmt.Sprintf("%s %s %s",
				statementStart,
				strings.TrimSuffix(
					strings.Repeat(fmt.Sprintf("(%s), ", helpers.QuestionMarks(parametersPerRecord)), recordCount),
					", ",
				),
				statementEnd,
			),
		)

		sgs.Logger.Debug("executing-batched-statement", lager.Data{"batch": batchNumber, "recordCount": recordCount, "paramCount": len(values)})
		_, err := tx.Exec(reboundStatement, values...)
		if err != nil {
			sgs.Logger.Error("batch-prepared-statement-failed", err, lager.Data{"query": reboundStatement})
			return fmt.Errorf("executing batched statement: %s", err)
		}
		batchNumber++
	}

	return nil
}

func (sgs *SGStore) ReplaceSecurityGroupSpaceAssociations(tx db.Transaction, table string, bindings map[string][]string) error {
	var deleteWhereBindings, insertValues []any
	var insertTxSize, deleteTxSize int
	for sgGuid, spaceGuids := range bindings {
		deleteWhereBindings = append(deleteWhereBindings, sgGuid)
		deleteTxSize += len(sgGuid)

		for _, spaceGuid := range spaceGuids {
			insertValues = append(insertValues, sgGuid, spaceGuid)
			insertTxSize += len(sgGuid) + len(spaceGuid)
		}
	}

	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE security_group_guid IN (", table)
	err := sgs.BatchPreparedStatement(tx, deleteQuery, ")", deleteWhereBindings, 1)
	if err != nil {
		return fmt.Errorf("deleting previous associations: %s", err)
	}

	replaceQuery := fmt.Sprintf("INSERT INTO %s (security_group_guid, space_guid) VALUES", table)
	columnsPerRecord := 2
	err = sgs.BatchPreparedStatement(tx, replaceQuery, "", insertValues, columnsPerRecord)
	if err != nil {
		return fmt.Errorf("creating new associations: %s", err)
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

func (sgs *SGStore) onConflictUpdateSQL(columns ...string) string {
	var conflictSql string
	switch sgs.Conn.DriverName() {
	case helpers.MySQL:
		conflictSql = "ON DUPLICATE KEY UPDATE"
		for _, column := range columns {
			conflictSql = fmt.Sprintf("%s %s = VALUES(%s),", conflictSql, column, column)
		}
		conflictSql = strings.TrimRight(conflictSql, ",")
	case helpers.Postgres:
		conflictSql = "ON CONFLICT (guid) DO UPDATE SET "
		for _, column := range columns {
			conflictSql = fmt.Sprintf("%s %s = EXCLUDED.%s,", conflictSql, column, column)
		}
		conflictSql = strings.TrimRight(conflictSql, ",")
	default:
		return ""
	}
	return conflictSql
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
