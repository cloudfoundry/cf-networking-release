package store

import (
	"encoding/json"
	"fmt"
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
	SpacesWithExpiredOrNoCache([]string) ([]string, error)
	UpdateSecurityGroupsFromCapi([]string) error
	LastUpdated() (int, error)
}

type SGStore struct {
	Logger      lager.Logger
	Conn        Database
	CacheExpiry time.Duration
	CCClient    cc_client.CCClient
	UAAClient   *uaa_client.Client
}

func (sgs *SGStore) UpdateSecurityGroupsFromCapi(spacesNeedingRefresh []string) error {
	tx, err := sgs.Conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}
	defer tx.Rollback()

	// FIXME: should this be cached and renewed vs every request?
	token, err := sgs.UAAClient.GetToken()
	if err != nil {
		return fmt.Errorf("get UAA token failed: %s", err)
	}

	securityGroups, err := sgs.CCClient.GetSecurityGroupsBySpaces(token, spacesNeedingRefresh)
	if err != nil {
		return err
	}

	securityGroupsForSpace := map[string]map[string]SecurityGroup{}
	for _, capiSg := range securityGroups {
		spacesForSg := map[string]struct{}{}
		var stagingSpaces, runningSpaces []string
		for _, spaces := range capiSg.Relationships.StagingSpaces.Data {
			for _, space := range spaces {
				spacesForSg[space] = struct{}{}
				stagingSpaces = append(stagingSpaces, space)
			}
		}
		for _, spaces := range capiSg.Relationships.RunningSpaces.Data {
			for _, space := range spaces {
				spacesForSg[space] = struct{}{}
				runningSpaces = append(runningSpaces, space)
			}
		}

		rules, err := json.Marshal(capiSg.Rules)
		if err != nil {
			return fmt.Errorf("error converting rules to json for ASG '%s': %s", capiSg.GUID, err)
		}
		sg := SecurityGroup{
			Guid:              capiSg.GUID,
			Name:              capiSg.Name,
			Rules:             string(rules),
			StagingDefault:    capiSg.GloballyEnabled.Staging,
			RunningDefault:    capiSg.GloballyEnabled.Running,
			StagingSpaceGuids: stagingSpaces,
			RunningSpaceGuids: runningSpaces,
		}

		for _, spaceGuid := range spacesNeedingRefresh {
			_, boundToSpace := spacesForSg[spaceGuid]
			if boundToSpace || sg.StagingDefault || sg.RunningDefault {
				sgs.Logger.Info("FIXME-adding-securitygroup-to-space", lager.Data{"security-group": sg.Name, "space": spaceGuid, "boundToSpace": boundToSpace, "stagingDefault": sg.StagingDefault, "runningDefault": sg.RunningDefault})
				if securityGroupsForSpace[spaceGuid] == nil {
					securityGroupsForSpace[spaceGuid] = map[string]SecurityGroup{}
				}
				securityGroupsForSpace[spaceGuid][sg.Guid] = sg
			} else {
				sgs.Logger.Info("FIXME-securitygroup-not-for-space", lager.Data{"space": spaceGuid, "asg": sg})
			}
		}
	}

	// FIXME: rename lastUpdated to expiresAt
	upsertQuery := tx.Rebind(`
		INSERT INTO spaces 
		(guid, lastUpdated, asgs)
		VALUES(?, ?, ?) ` +
		sgs.onConflictUpdateSQL() +
		` lastUpdated=?, asgs=?`)
	lastUpdated := time.Now()

	for spaceGuid, sgMap := range securityGroupsForSpace {
		var sGroups SecurityGroups
		for _, sg := range sgMap {
			sGroups = append(sGroups, sg)
		}

		sgs.Logger.Info("FIXME-about-to-cache-space", lager.Data{"space": spaceGuid, "asgs": sGroups, "query": upsertQuery})
		_, err = tx.Exec(upsertQuery, spaceGuid, lastUpdated, sGroups, lastUpdated, sGroups)
		if err != nil {
			return fmt.Errorf("failed updating security group cache for space %s: %s", spaceGuid, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}
	return nil
}

// FIXME: purge expired Space caches?

func (sgs *SGStore) SpacesWithExpiredOrNoCache(spaceGuids []string) ([]string, error) {
	sgs.Logger.Info("FIXME-checking-for-expired-cache-entries")
	query := `SELECT guid, lastUpdated FROM spaces`
	if len(spaceGuids) > 0 {
		whereClause := fmt.Sprintf("guid IN (%s)",
			helpers.QuestionMarks(len(spaceGuids)),
		)
		query = fmt.Sprintf("%s WHERE %s", query, whereClause)
	}

	var whereBindings []interface{}
	for _, guid := range spaceGuids {
		whereBindings = append(whereBindings, guid)
	}

	rebindedQuery := helpers.RebindForSQLDialectAndMark(query, sgs.Conn.DriverName(), "%")
	// FIXME: do the time math in sql?

	rows, err := sgs.Conn.Query(rebindedQuery, whereBindings...)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	var expiredOrNotCachedSpaces []string
	spacesSeen := map[string]struct{}{}

	for rows.Next() {
		var guid string
		var lastUpdated time.Time
		err = rows.Scan(&guid, &lastUpdated)
		if err != nil {
			return []string{}, err
		}

		expiresAt := lastUpdated.Add(sgs.CacheExpiry)
		if time.Now().After(expiresAt) {
			sgs.Logger.Info("FIXME-space-cache-expired", lager.Data{"space": guid, "expiry": expiresAt})
			expiredOrNotCachedSpaces = append(expiredOrNotCachedSpaces, guid)
		}
		spacesSeen[guid] = struct{}{}
	}

	for _, guid := range spaceGuids {
		if _, ok := spacesSeen[guid]; !ok {
			sgs.Logger.Info("FIXME-no-cache-yet-exists-for-space", lager.Data{"space": guid})
			expiredOrNotCachedSpaces = append(expiredOrNotCachedSpaces, guid)
		}
	}

	return expiredOrNotCachedSpaces, nil
}

func (sgs *SGStore) BySpaceGuids(spaceGuids []string, page Page) ([]SecurityGroup, Pagination, error) {
	query := `
		SELECT
			id,
			guid,
			asgs
		FROM spaces`

	var whereClause string
	if len(spaceGuids) > 0 {
		whereClause = fmt.Sprintf("(guid IN (%s))",
			helpers.QuestionMarks(len(spaceGuids)),
		)
	}
	query = fmt.Sprintf("%s WHERE %s", query, whereClause)

	whereBindings := make([]any, len(spaceGuids))
	for i, spaceGuid := range spaceGuids {
		whereBindings[i] = spaceGuid
	}

	if page.From > 0 {
		query = query + " AND id >= ?"
		whereBindings = append(whereBindings, page.From)
	}
	query = query + " ORDER BY id"

	if page.Limit > 0 {
		// we don't use a placeholder because limit is an integer and it is safe to interpolate it
		query = fmt.Sprintf(`%s LIMIT %d`, query, page.Limit+1)
	}

	rebindedQuery := helpers.RebindForSQLDialectAndMark(query, sgs.Conn.DriverName(), "%")

	sgs.Logger.Info("FIXME-by-spqce-guids-query", lager.Data{"query": rebindedQuery, "bindings": whereBindings})
	rows, err := sgs.Conn.Query(rebindedQuery, whereBindings...)
	if err != nil {
		return nil, Pagination{}, fmt.Errorf("selecting security groups: %s", err)
	}
	defer rows.Close()

	result := map[string]SecurityGroup{}
	nextId := 0
	for rows.Next() {
		var id int
		var spaceCache SpaceCache
		err := rows.Scan(&id,
			&spaceCache.Guid,
			&spaceCache.ASGs,
		)
		if err != nil {
			return nil, Pagination{}, fmt.Errorf("scanning security group result: %s", err)
		}

		if page.Limit == 0 || len(result) < page.Limit {
			for _, sg := range spaceCache.ASGs {
				result[sg.Guid] = sg
			}
		} else {
			nextId = id
		}
	}
	var asgs []SecurityGroup
	for _, v := range result {
		asgs = append(asgs, v)
	}

	return asgs, Pagination{Next: nextId}, nil
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
