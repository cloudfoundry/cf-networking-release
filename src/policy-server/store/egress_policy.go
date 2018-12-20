package store

import (
	"database/sql"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

type EgressPolicyTable struct {
	Conn  Database
	Guids guidGenerator
}

func (e *EgressPolicyTable) CreateApp(tx db.Transaction, sourceTerminalGUID, appGUID string) error {
	_, err := tx.Exec(tx.Rebind(`INSERT INTO apps (terminal_guid, app_guid) VALUES (?,?)`),
		sourceTerminalGUID,
		appGUID,
	)
	return err
}

func (e *EgressPolicyTable) CreateDefault(tx db.Transaction, sourceTerminalGUID string) error {
	_, err := tx.Exec(tx.Rebind(`INSERT INTO defaults (terminal_guid) VALUES (?)`), sourceTerminalGUID)
	return err
}

func (e *EgressPolicyTable) CreateEgressPolicy(tx db.Transaction, sourceTerminalGUID, destinationTerminalGUID, appLifecycle string) (string, error) {
	guid := e.Guids.New()

	_, err := tx.Exec(tx.Rebind(`
			INSERT INTO egress_policies (guid, source_guid, destination_guid, app_lifecycle)
			VALUES (?,?,?,?)
		`),
		guid,
		sourceTerminalGUID,
		destinationTerminalGUID,
		appLifecycle,
	)

	if err != nil {
		return "", fmt.Errorf("error inserting egress policy: %s", err)
	}

	return guid, nil
}

func (e *EgressPolicyTable) CreateSpace(tx db.Transaction, sourceTerminalGUID, spaceGUID string) error {
	_, err := tx.Exec(tx.Rebind(`INSERT INTO spaces (terminal_guid, space_guid) VALUES (?,?)`),
		sourceTerminalGUID,
		spaceGUID,
	)
	return err
}

func (e *EgressPolicyTable) DeleteEgressPolicy(tx db.Transaction, egressPolicyGUID string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM egress_policies WHERE guid = ?`), egressPolicyGUID)
	return err
}

func (e *EgressPolicyTable) DeleteIPRange(tx db.Transaction, ipRangeID int64) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM ip_ranges WHERE id = ?`), ipRangeID)
	return err
}

func (e *EgressPolicyTable) DeleteApp(tx db.Transaction, terminalGUID string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM apps WHERE terminal_guid = ?`), terminalGUID)
	return err
}

func (e *EgressPolicyTable) DeleteDefault(tx db.Transaction, terminalGUID string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM defaults WHERE terminal_guid = ?`), terminalGUID)
	return err
}

func (e *EgressPolicyTable) DeleteSpace(tx db.Transaction, terminalGUID string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM spaces WHERE terminal_guid = ?`), terminalGUID)
	return err
}

func (e *EgressPolicyTable) IsTerminalInUse(tx db.Transaction, terminalGUID string) (bool, error) {
	var count int64
	err := tx.QueryRow(tx.Rebind(`SELECT COUNT(guid) FROM egress_policies WHERE source_guid = ? OR destination_guid = ?`), terminalGUID, terminalGUID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (e *EgressPolicyTable) GetByGUID(tx db.Transaction, guids ...string) ([]EgressPolicy, error) {
	if len(guids) == 0 {
		return []EgressPolicy{}, nil
	}

	rows, err := tx.Queryx(tx.Rebind(
		selectEgressPolicyQuery(`egress_policies.guid IN (`+generateQuestionMarkString(len(guids))+`)`)),
		convertToInterfaceSlice(guids)...)
	if err != nil {
		return []EgressPolicy{}, err
	}

	return e.convertRowsToEgressPolicies(rows)
}

func (e *EgressPolicyTable) GetTerminalByAppGUID(tx db.Transaction, appGUID string) (string, error) {
	var guid string

	err := tx.QueryRow(tx.Rebind(`SELECT terminal_guid FROM apps WHERE app_guid = ?`),
		appGUID,
	).Scan(&guid)

	if err != nil && err == sql.ErrNoRows {
		return "", nil
	} else {
		return guid, err
	}
}

func (e *EgressPolicyTable) GetTerminalBySpaceGUID(tx db.Transaction, spaceGUID string) (string, error) {
	var guid string

	err := tx.QueryRow(tx.Rebind(`SELECT terminal_guid FROM spaces WHERE space_guid = ?`),
		spaceGUID,
	).Scan(&guid)

	if err != nil && err == sql.ErrNoRows {
		return "", nil
	} else {
		return guid, err
	}
}

func (e *EgressPolicyTable) GetAllPolicies() ([]EgressPolicy, error) {
	rows, err := e.Conn.Query(selectEgressPolicyQuery())
	if err != nil {
		return []EgressPolicy{}, err
	}

	return e.convertRowsToEgressPolicies(rows)
}

func (e *EgressPolicyTable) GetBySourceGuidsAndDefaults(ids []string) ([]EgressPolicy, error) {

	query := selectEgressPolicyQuery(fmt.Sprintf(`
		apps.app_guid IN (%[1]s) OR spaces.space_guid IN (%[1]s) OR defaults.terminal_guid IS NOT NULL
		`, generateQuestionMarkString(len(ids))))

	ids = append(ids, ids...)
	rows, err := e.Conn.Query(e.Conn.Rebind(query), convertToInterfaceSlice(ids)...)
	if err != nil {
		return []EgressPolicy{}, err
	}

	return e.convertRowsToEgressPolicies(rows)
}

func (e *EgressPolicyTable) GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles []string) ([]EgressPolicy, error) {
	var whereClauses []string

	if len(sourceIds) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf(`(apps.app_guid IN (%[1]s) OR spaces.space_guid IN (%[1]s))`, generateQuestionMarkString(len(sourceIds))))
	}

	if len(sourceTypes) > 0 {
		for _, sourceType := range sourceTypes {
			if sourceType == "app" {
				whereClauses = append(whereClauses, "spaces.space_guid IS NULL")
			} else {
				whereClauses = append(whereClauses, "apps.app_guid IS NULL")
			}
		}
	}

	if len(destinationIds) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf(`ip_ranges.terminal_guid IN (%[1]s)`, generateQuestionMarkString(len(destinationIds))))
	}

	if len(destinationNames) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf(`destination_metadatas.name IN (%[1]s)`, generateQuestionMarkString(len(destinationNames))))
	}

	if len(appLifecycles) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf(`egress_policies.app_lifecycle IN (%[1]s)`, generateQuestionMarkString(len(appLifecycles))))
	}

	query := selectEgressPolicyQuery(whereClauses...)

	sourceIds = append(sourceIds, sourceIds...)
	sourceIds = append(sourceIds, destinationIds...)
	sourceIds = append(sourceIds, destinationNames...)
	sourceIds = append(sourceIds, appLifecycles...)

	rows, err := e.Conn.Query(e.Conn.Rebind(query), convertToInterfaceSlice(sourceIds)...)
	if err != nil {
		return []EgressPolicy{}, err
	}

	return e.convertRowsToEgressPolicies(rows)
}

func selectEgressPolicyQuery(extraClauses ...string) string {
	extraClauses = append(extraClauses, "egress_policies.guid IS NOT NULL")

	return fmt.Sprintf(`
		SELECT
			egress_policies.guid,
			egress_policies.source_guid,
			egress_policies.app_lifecycle,
			COALESCE(destination_metadatas.name, ''),
			COALESCE(destination_metadatas.description, ''),
			apps.app_guid,
			spaces.space_guid,
			defaults.terminal_guid,
			ip_ranges.terminal_guid,
			ip_ranges.protocol,
			ip_ranges.start_ip,
			ip_ranges.end_ip,
			ip_ranges.start_port,
			ip_ranges.end_port,
			ip_ranges.icmp_type,
			ip_ranges.icmp_code,
			ip_ranges.description
		FROM egress_policies
		LEFT OUTER JOIN ip_ranges ON (ip_ranges.terminal_guid = egress_policies.destination_guid)
		LEFT OUTER JOIN apps ON (egress_policies.source_guid = apps.terminal_guid)
		LEFT OUTER JOIN spaces ON (egress_policies.source_guid = spaces.terminal_guid)
		LEFT OUTER JOIN defaults ON (egress_policies.source_guid = defaults.terminal_guid)
		LEFT OUTER JOIN destination_metadatas ON (egress_policies.destination_guid = destination_metadatas.terminal_guid)
		WHERE %s
		ORDER BY ip_ranges.id;`, strings.Join(extraClauses, " AND "))
}

type sqlRows interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
}

func (e *EgressPolicyTable) convertRowsToEgressPolicies(rows sqlRows) ([]EgressPolicy, error) {
	foundPolicieIdxs := make(map[string]int)
	var policiesToReturn []EgressPolicy
	var counter int
	defer rows.Close()
	for rows.Next() {
		var egressPolicyGUID, sourceTerminalGUID, appLifecycle, name, description, destinationGUID, sourceAppGUID, sourceSpaceGUID, defaultTerminalGUID, protocol, startIP, endIP, ruleDescription *string
		var startPort, endPort, icmpType, icmpCode int
		err := rows.Scan(
			&egressPolicyGUID,
			&sourceTerminalGUID,
			&appLifecycle,
			&name,
			&description,
			&sourceAppGUID,
			&sourceSpaceGUID,
			&defaultTerminalGUID,
			&destinationGUID,
			&protocol,
			&startIP,
			&endIP,
			&startPort,
			&endPort,
			&icmpType,
			&icmpCode,
			&ruleDescription)
		if err != nil {
			return []EgressPolicy{}, err
		}

		var ports []Ports
		if startPort != 0 && endPort != 0 {
			ports = []Ports{
				{
					Start: startPort,
					End:   endPort,
				},
			}
		}

		foundIndex, ok := foundPolicieIdxs[*egressPolicyGUID]
		if ok {
			policy := policiesToReturn[foundIndex]
			policy.Destination.Rules = append(policy.Destination.Rules, EgressDestinationRule{
				Protocol: *protocol,
				Ports:    ports,
				IPRanges: []IPRange{{Start: *startIP, End: *endIP}},
				ICMPType: icmpType,
				ICMPCode: icmpCode,
				Description: *ruleDescription,
			})
			policiesToReturn[foundIndex] = policy
		} else {
			var source EgressSource

			switch {
			case sourceSpaceGUID != nil:
				source = EgressSource{
					ID:           *sourceSpaceGUID,
					Type:         "space",
					TerminalGUID: *sourceTerminalGUID,
				}
			case sourceSpaceGUID == nil && sourceAppGUID == nil:
				source = EgressSource{
					Type:         "default",
					TerminalGUID: *sourceTerminalGUID,
				}
			default:
				source = EgressSource{
					ID:           *sourceAppGUID,
					Type:         "app",
					TerminalGUID: *sourceTerminalGUID,
				}
			}

			egressPolicy := EgressPolicy{
				ID:     *egressPolicyGUID,
				Source: source,
				Destination: EgressDestination{
					GUID:        *destinationGUID,
					Name:        *name,
					Description: *description,
					Rules: []EgressDestinationRule{
						{
							Protocol: *protocol,
							Ports:    ports,
							IPRanges: []IPRange{
								{
									Start: *startIP,
									End:   *endIP,
								},
							},
							ICMPType: icmpType,
							ICMPCode: icmpCode,
							Description: *ruleDescription,
						},
					},
				},
				AppLifecycle: *appLifecycle,
			}
			policiesToReturn = append(policiesToReturn, egressPolicy)
			foundPolicieIdxs[*egressPolicyGUID] = counter
			counter = counter + 1
		}
	}

	return policiesToReturn, nil
}
