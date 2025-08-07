package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/store/helpers"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

//counterfeiter:generate -o fakes/security_groups_store.go --fake-name SecurityGroupsStore . SecurityGroupsStore
type SecurityGroupsStore interface {
	Replace([]SecurityGroup) error
	BySpaceGuids([]string, Page) ([]SecurityGroup, Pagination, error)
	UpdateSpaceCache(SpaceCache) error
	LastUpdated() (int, error)
	CheckForASGUpdates([]string, time.Time) (bool, error)
}

type SpaceCache struct {
	Spaces map[string]Space
}

type Space struct {
	Id          int
	Guid        string
	LastUpdated time.Time
	ASGs        map[string]SecurityGroup
	Hash        string
}

type SGStore struct {
	Logger      lager.Logger
	Conn        Database
	CacheExpiry time.Duration
	CCClient    cc_client.CCClient
	UAAClient   *uaa_client.Client
}

func (sgs *SGStore) UpdateSpaceCache(spaceCache SpaceCache) error {
	if len(spaceCache.Spaces) == 0 {
		return fmt.Errorf("No spaces provided to update from")
	}
	sgs.Logger.Info("FIXME-gathering-space-cache-hashes")
	existingSpaces := map[string]string{}
	rows, err := sgs.Conn.Query("SELECT guid, hash FROM spaces")
	if err != nil {
		return fmt.Errorf("failed-reading-sg-hashes-for-spaces: %s", err)
	}
	defer rows.Close()
	for rows.Next() {
		var guid string
		var hash sql.NullString
		err := rows.Scan(&guid, &hash)
		if err != nil {
			return fmt.Errorf("failed-to-scan-sg-hash-for-space: %s", err)
		}
		existingSpaces[guid] = hash.String
	}
	sgs.Logger.Info("FIXME-gathering-space-cache-hashes-done")

	lastUpdated := time.Now()

	seen := map[string]struct{}{}
	upsertQueryBeginning := `
		INSERT INTO spaces 
		(guid, hash, lastUpdated, asgs)
		VALUES `
	upsertQueryEnd := sgs.onConflictUpdateSQL("hash", "lastUpdated", "asgs")
	upsertQueryString := upsertQueryBeginning
	queryLen := len(upsertQueryString) + len(upsertQueryEnd)
	whereBindings := []any{}
	var currentlyExecutingTransactionId string

	var maxTransactionLength int
	var unused string
	err = sgs.Conn.QueryRow("SHOW VARIABLES LIKE 'wsrep_max_ws_size'").Scan(&unused, &maxTransactionLength)
	if err != nil {
		sgs.Logger.Error("FIXME-failed-to-get-wsrep-max-size-assuming-it-is-within-one-trnasaction", err)
	}
	sgs.Logger.Info("FIXME-wsrep-max-size", lager.Data{"wsrep_max_ws_size": maxTransactionLength})

	n := 0

	sgs.Logger.Info("FIXME-looping-through-spaces")
	for _, space := range spaceCache.Spaces {
		oldHash := existingSpaces[space.Guid]
		delete(existingSpaces, space.Guid)
		var sGroups SecurityGroups
		for _, sg := range space.ASGs {
			sGroups = append(sGroups, sg)
		}

		slices.SortFunc(sGroups, func(a, b SecurityGroup) int {
			return strings.Compare(strings.ToLower(a.Guid), strings.ToLower(b.Guid))
		})

		sGroupsJson, err := json.Marshal(sGroups)
		if err != nil {
			return fmt.Errorf("failed-marshalling-ASGs-to-json: %s", err)
		}

		seen[space.Guid] = struct{}{}
		newHash := fmt.Sprintf("%x", sha256.Sum256(sGroupsJson))

		if newHash != oldHash {
			if n != 1 {
				n = 1
				var oldRecord []byte
				err = sgs.Conn.QueryRow("SELECT asgs FROM spaces WHERE guid = ?", space.Guid).Scan(&oldRecord)
				if err != nil {
					sgs.Logger.Error("FIMXE-failed-to-get-old-space-sg-json", err)
				}
			}
			nextRecordLength := lengthOfNextRecord(space.Guid, newHash, lastUpdated, sGroupsJson)
			// maxTransactionLength will be set to either 0 (postgres, non-galera mysql), or the transaction limit
			// if no limit was detectable, try everything in a single transaction
			if maxTransactionLength != 0 && queryLen+nextRecordLength >= maxTransactionLength*25/100 {
				upsertQueryString = strings.TrimRight(upsertQueryString, ",") + sgs.onConflictUpdateSQL("hash", "lastUpdated", "asgs")
				currentlyExecutingTransactionId, err = sgs.batchInsertSpaceRecords(upsertQueryString, whereBindings, currentlyExecutingTransactionId, queryLen)
				if err != nil {
					return err
				}
				upsertQueryString = upsertQueryBeginning
				whereBindings = []any{}
				queryLen = len(upsertQueryString) + len(upsertQueryEnd)
			}
			queryLen += nextRecordLength
			upsertQueryString = upsertQueryString + ` (?, ?, ?, ?),`
			whereBindings = append(whereBindings, space.Guid, newHash, lastUpdated, sGroupsJson)
		}
	}
	if len(whereBindings) != 0 {
		upsertQueryString = strings.TrimRight(upsertQueryString, ",") + sgs.onConflictUpdateSQL("hash", "lastUpdated", "asgs")
		_, err = sgs.batchInsertSpaceRecords(upsertQueryString, whereBindings, currentlyExecutingTransactionId, queryLen)
		if err != nil {
			return err
		}
	}

	// at this point, existingSpaces should only consist of spaces retrieved from the DB, but not seen in the data just synced up
	err = sgs.deleteStaleRecordsByGuid("spaces", existingSpaces)
	if err != nil {
		return err
	}

	return nil
}

func (sgs *SGStore) deleteStaleRecordsByGuid(table string, argMap map[string]string) error {
	sgs.Logger.Info("FIXME-trying-to-delete-records")
	if len(argMap) > 0 {
		tx, err := sgs.Conn.Beginx()
		if err != nil {
			sgs.Logger.Error("failed-creating-transaction", err)
			return fmt.Errorf("create transaction: %s", err)
		}
		defer tx.Rollback()

		deleteQuery := tx.Rebind(fmt.Sprintf("DELETE FROM %s WHERE guid IN (%s)", table, helpers.QuestionMarks(len(argMap))))

		args := []any{}
		for guid := range argMap {
			args = append(args, guid)
		}
		sgs.Logger.Info("FIXME-trying-to-delete-records", lager.Data{table: args})
		_, err = tx.Exec(deleteQuery, args...)
		if err != nil {
			sgs.Logger.Error("failed-deleting-records-no-longer-present-in-capi", err, lager.Data{table: args})
			return err
		}
		sgs.Logger.Info("FIXME-committing-transaction")
		err = tx.Commit()
		if err != nil {
			sgs.Logger.Error("failed-committing-transaction", err)
			return fmt.Errorf("committing transaction: %s", err)
		}
	}
	return nil
}

func (sgs *SGStore) batchInsertSpaceRecords(queryString string, whereBindings []any, previousTransactionID string, transactionLength int) (string, error) {
	sgs.Logger.Info("FIXME-executing-batch-insert", lager.Data{"record_count": len(whereBindings) / 4, "transaction_length": transactionLength})
	defer sgs.Logger.Info("FIXME-executing-batch-done")

	tx, err := sgs.Conn.Beginx()
	if err != nil {
		sgs.Logger.Error("failed-creating-transaction", err)
		return previousTransactionID, fmt.Errorf("create transaction: %s", err)
	}
	defer tx.Rollback()

	if previousTransactionID != "" {
		sgs.Logger.Info("FIXME-waiting-for-previous-insert-to-complete", lager.Data{"transaction_id": previousTransactionID})
		waitQuery := tx.Rebind(`SELECT WSREP_SYNC_WAIT_UPTO_GTID(?)`)
		_, err := tx.Exec(waitQuery, previousTransactionID)
		if err != nil {
			sgs.Logger.Error("failed-waiting-for-previous-insert-transaction-to-finish", err, lager.Data{"transaction_id": previousTransactionID})
		}
	}
	query := tx.Rebind(queryString)
	_, err = tx.Exec(query, whereBindings...)
	if err != nil {
		sgs.Logger.Error("failed-updating-space-cache", err)
		return previousTransactionID, fmt.Errorf("failed updating space cache: %s", err)
	}
	err = tx.Commit()
	if err != nil {
		sgs.Logger.Error("failed-committing-transaction", err)
		return previousTransactionID, fmt.Errorf("committing transaction: %s", err)
	}

	var currentTransactionId string
	err = sgs.Conn.QueryRow(`SELECT WSREP_LAST_WRITTEN_GTID()`).Scan(&currentTransactionId)
	if err != nil {
		sgs.Logger.Error("failed-to-find-current-transaction-id-will-execute-next-query-immediately", err)
	}
	return currentTransactionId, nil
}

func lengthOfNextRecord(newArgs ...any) int {
	newLength := 2 // " ("
	for _, arg := range newArgs {
		// FIXME: make this work for all data types
		switch typedArg := arg.(type) {
		case string:
			newLength += len(typedArg)
		case time.Time:
			// format: 2006-01-02 15:04:05
			newLength += 19
		case []byte:
			newLength += len(typedArg)
		default:
			newLength += len(fmt.Sprintf("%v", typedArg))
		}

		newLength += 3 // "?, "
	}
	newLength -= 2 // remove the last 2 characters from ^^ since it doesn't end on ', '
	newLength += 3 // "), "
	return newLength
}

func (sgs SGStore) CheckForASGUpdates(spaceGuids []string, since time.Time) (bool, error) {
	globalLastUpdated, err := sgs.LastUpdated()
	if err != nil {
		return false, fmt.Errorf("failed to get last updated time: %s", err)
	}
	if time.UnixMicro(int64(globalLastUpdated)).After(since) {
		return true, nil
	}

	query := `SELECT COUNT(guid) FROM spaces`
	var whereClause string
	whereBindings := []any{}

	if len(spaceGuids) > 0 {
		whereClause = fmt.Sprintf("(guid IN (%s)) AND",
			helpers.QuestionMarks(len(spaceGuids)),
		)
		for _, spaceGuid := range spaceGuids {
			whereBindings = append(whereBindings, spaceGuid)
		}
	}

	whereClause = whereClause + " lastUpdated > ?"
	whereBindings = append(whereBindings, since)

	query = fmt.Sprintf("%s WHERE %s", query, whereClause)

	sgs.Logger.Info("FIXME-check-for-asg-updates", lager.Data{"query": query, "bindings": whereBindings})
	rows, err := sgs.Conn.Query(query, whereBindings...)
	if err != nil {
		return false, fmt.Errorf("selecting security groups: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		var count int
		err := rows.Scan(&count)
		if err != nil {
			return false, fmt.Errorf("scanning security group result: %s", err)
		}

		sgs.Logger.Info("FIXME-asgs-with-updates", lager.Data{"count-of-updated-asgs": count})
		if count > 0 {
			return true, nil
		}
	}

	return false, nil
}

func (sgs *SGStore) BySpaceGuids(spaceGuids []string, page Page) ([]SecurityGroup, Pagination, error) {
	nextId := 0
	result := map[string]SecurityGroup{}
	if len(spaceGuids) > 0 {
		query := `
		SELECT
			id,
			guid,
			asgs
		FROM spaces`

		whereClause := " WHERE"
		whereClause = fmt.Sprintf("%s (guid IN (%s))", whereClause,
			helpers.QuestionMarks(len(spaceGuids)),
		)

		whereBindings := make([]any, len(spaceGuids))
		for i, spaceGuid := range spaceGuids {
			whereBindings[i] = spaceGuid
		}

		if page.From > 0 {
			whereClause = whereClause + " AND id >= ?"
			whereBindings = append(whereBindings, page.From)
		}
		if page.From > 0 || len(spaceGuids) > 0 {
			query = fmt.Sprintf("%s%s", query, whereClause)
		}
		query = query + " ORDER BY id"

		if page.Limit > 0 {
			// we don't use a placeholder because limit is an integer and it is safe to interpolate it
			query = fmt.Sprintf(`%s LIMIT %d`, query, page.Limit+1)
		}

		rebindedQuery := helpers.RebindForSQLDialectAndMark(query, sgs.Conn.DriverName(), "%")

		sgs.Logger.Info("FIXME-by-space-guids-query", lager.Data{"query": rebindedQuery, "bindings": whereBindings})
		rows, err := sgs.Conn.Query(rebindedQuery, whereBindings...)
		if err != nil {
			return nil, Pagination{}, fmt.Errorf("selecting security groups: %s", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var spaceGuid string
			var sgroups SecurityGroups
			err := rows.Scan(&id,
				&spaceGuid,
				&sgroups,
			)
			if err != nil {
				return nil, Pagination{}, fmt.Errorf("scanning security group result: %s", err)
			}
			sgs.Logger.Info("FIXME-retrieved-space-from-store", lager.Data{"guid": spaceGuid})

			if page.Limit == 0 || len(result) < page.Limit {
				for _, sg := range sgroups {
					sgs.Logger.Info("FIXME-found-asg", lager.Data{"asg": sg.Guid})
					result[sg.Guid] = sg
				}
			} else {
				nextId = id
				break
			}
		}
	}

	// FIXME: figure out how to deal with pagination
	query := `
		SELECT guid, name, rules, staging_default, running_default, staging_spaces, running_spaces
		FROM security_groups
		WHERE running_default = true OR staging_default = true`

	sgs.Logger.Info("FIXME-by-space-guids-global-sgs-query", lager.Data{"query": query})
	sgRows, err := sgs.Conn.Query(query)
	if err != nil {
		return nil, Pagination{}, fmt.Errorf("selecting global security groups: %s", err)
	}
	defer sgRows.Close()

	for sgRows.Next() {
		var guid, name, rules string
		var stagingDefault, runningDefault bool
		var stagingSpaceGuids, runningSpaceGuids SpaceGuids
		err := sgRows.Scan(&guid, &name, &rules, &stagingDefault, &runningDefault, &stagingSpaceGuids, &runningSpaceGuids)
		if err != nil {
			return nil, Pagination{}, fmt.Errorf("error scanning global security group: %s", err)
		}

		result[guid] = SecurityGroup{
			Guid:              guid,
			Name:              name,
			Rules:             rules,
			StagingDefault:    stagingDefault,
			RunningDefault:    runningDefault,
			StagingSpaceGuids: stagingSpaceGuids,
			RunningSpaceGuids: runningSpaceGuids,
		}
	}
	var asgs []SecurityGroup
	for _, v := range result {
		asgs = append(asgs, v)
	}
	sgs.Logger.Info("FIXME-by-space-guids-asgs-retrieved", lager.Data{"asgs": asgs})

	return asgs, Pagination{Next: nextId}, nil
}

func (sgs *SGStore) Replace(newSecurityGroups []SecurityGroup) error {
	tx, err := sgs.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}
	defer tx.Rollback()

	existingGuids := map[string]string{}
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
			existingGuids[guid] = guid
		}
	}
	sgs.Logger.Info("FIXME-grabbed-global-security-group-guids")

	upsertQuery := tx.Rebind(`
		INSERT INTO security_groups
		(guid, name, rules, staging_default, running_default, staging_spaces, running_spaces)
		VALUES(?, ?, ?, ?, ?, ?, ?) ` +
		sgs.onConflictUpdateSQL("name", "rules", "staging_default", "running_default", "staging_spaces", "running_spaces"))

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
		)
		if err != nil {
			return fmt.Errorf("saving security group %s (%s): %s", group.Guid, group.Name, err)
		}
	}
	sgs.Logger.Info("FIXME-updated-global-security-groups")

	err = sgs.updateLastUpdated(tx)
	if err != nil {
		return fmt.Errorf("updating security_groups_info.last_updated: %s", err)
	}
	sgs.Logger.Info("FIXME-updated-last-updated")

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	err = sgs.deleteStaleRecordsByGuid("security_groups", existingGuids)
	if err != nil {
		return fmt.Errorf("deleting security groups: %s", err)
	}
	sgs.Logger.Info("FIXME-deleted-global-security-groups")

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
