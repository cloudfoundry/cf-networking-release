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

func (e *EgressPolicyTable) CreateApp(tx db.Transaction, sourceTerminalGUID, appGUID string) (int64, error) {
	driverName := tx.DriverName()

	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO apps (terminal_guid, app_guid)
			VALUES (?,?)
		`),
			sourceTerminalGUID,
			appGUID,
		)
		if err != nil {
			return -1, err
		}

		return result.LastInsertId()
	} else if driverName == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO apps (terminal_guid, app_guid)
			VALUES (?,?)
			RETURNING id
		`),
			sourceTerminalGUID,
			appGUID,
		).Scan(&id)

		if err != nil {
			return -1, fmt.Errorf("error inserting app: %s", err)
		}

		return id, nil
	}
	return -1, fmt.Errorf("unknown driver: %s", driverName)
}

func (e *EgressPolicyTable) CreateIPRange(tx db.Transaction, destinationTerminalGUID, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) (int64, error) {
	driverName := tx.DriverName()
	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_guid, start_port, end_port, icmp_type, icmp_code)
			VALUES (?,?,?,?,?,?,?,?)
		`),
			protocol,
			startIP,
			endIP,
			destinationTerminalGUID,
			startPort,
			endPort,
			icmpType,
			icmpCode,
		)

		if err != nil {
			return -1, fmt.Errorf("error inserting ip ranges: %s", err)
		}

		return result.LastInsertId()
	} else if driverName == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_guid, start_port, end_port, icmp_type, icmp_code)
			VALUES (?,?,?,?,?,?,?,?)
			RETURNING id
		`),
			protocol,
			startIP,
			endIP,
			destinationTerminalGUID,
			startPort,
			endPort,
			icmpType,
			icmpCode,
		).Scan(&id)

		if err != nil {
			return -1, fmt.Errorf("error inserting ip ranges: %s", err)
		}

		return id, nil
	}

	return -1, fmt.Errorf("unknown driver: %s", driverName)
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

func (e *EgressPolicyTable) CreateSpace(tx db.Transaction, sourceTerminalGUID, spaceGUID string) (int64, error) {
	driverName := tx.DriverName()

	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO spaces (terminal_guid, space_guid)
			VALUES (?,?)
		`),
			sourceTerminalGUID,
			spaceGUID,
		)
		if err != nil {
			return -1, err
		}

		return result.LastInsertId()
	} else if driverName == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO spaces (terminal_guid, space_guid)
			VALUES (?,?)
			RETURNING id
		`),
			sourceTerminalGUID,
			spaceGUID,
		).Scan(&id)

		if err != nil {
			return -1, fmt.Errorf("error inserting space: %s", err)
		}

		return id, nil
	}
	return -1, fmt.Errorf("unknown driver: %s", driverName)
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
		selectEgressPolicyQuery(`
			WHERE egress_policies.guid IN (`+generateQuestionMarkString(len(guids))+`)
			ORDER BY ip_ranges.id;`,
		)),
		convertToInterfaceSlice(guids)...)
	if err != nil {
		return []EgressPolicy{}, err
	}

	return e.convertRowsToEgressPolicies(rows)
}

func (e *EgressPolicyTable) GetTerminalByAppGUID(tx db.Transaction, appGUID string) (string, error) {
	var guid string

	err := tx.QueryRow(tx.Rebind(`
	SELECT terminal_guid FROM apps WHERE app_guid = ?
	`),
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

	err := tx.QueryRow(tx.Rebind(`
		SELECT terminal_guid FROM spaces WHERE space_guid = ?
	`),
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

func (e *EgressPolicyTable) GetBySourceGuids(ids []string) ([]EgressPolicy, error) {

	query := selectEgressPolicyQuery(fmt.Sprintf(`
		WHERE apps.app_guid IN (%[1]s) OR spaces.space_guid IN (%[1]s)
		ORDER BY ip_ranges.id;`, generateQuestionMarkString(len(ids))))

	ids = append(ids, ids...)
	rows, err := e.Conn.Query(e.Conn.Rebind(query), convertToInterfaceSlice(ids)...)
	if err != nil {
		return []EgressPolicy{}, err
	}

	return e.convertRowsToEgressPolicies(rows)
}

func (e *EgressPolicyTable) GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames []string) ([]EgressPolicy, error) {
	query := "WHERE "

	if len(sourceIds) > 0 {
		query += fmt.Sprintf(`(apps.app_guid IN (%[1]s) OR spaces.space_guid IN (%[1]s)) AND `, generateQuestionMarkString(len(sourceIds)))
	}

	if len(sourceTypes) > 0 {
		for _, sourceType := range sourceTypes {
			if sourceType == "app" {
				query += "spaces.space_guid IS NULL AND\n"
			} else {
				query += "apps.app_guid IS NULL AND\n"
			}
		}
	}

	if len(destinationIds) > 0 {
		query += fmt.Sprintf(`ip_ranges.terminal_guid IN (%[1]s) AND `, generateQuestionMarkString(len(destinationIds)))
	}

	if len(destinationNames) > 0 {
		query += fmt.Sprintf(`destination_metadatas.name IN (%[1]s) AND `, generateQuestionMarkString(len(destinationNames)))
	}

	query = selectEgressPolicyQuery(query + " 1=1 ORDER BY ip_ranges.id;")

	sourceIds = append(sourceIds, sourceIds...)
	sourceIds = append(sourceIds, destinationIds...)
	sourceIds = append(sourceIds, destinationNames...)

	rows, err := e.Conn.Query(e.Conn.Rebind(query), convertToInterfaceSlice(sourceIds)...)
	if err != nil {
		return []EgressPolicy{}, err
	}

	return e.convertRowsToEgressPolicies(rows)
}

func selectEgressPolicyQuery(extraClauses ...string) string {
	return fmt.Sprintf(`
		SELECT
			egress_policies.guid,
			egress_policies.source_guid,
			egress_policies.app_lifecycle,
			COALESCE(destination_metadatas.name, ''),
			COALESCE(destination_metadatas.description, ''),
			apps.app_guid,
			spaces.space_guid,
			ip_ranges.terminal_guid,
			ip_ranges.protocol,
			ip_ranges.start_ip,
			ip_ranges.end_ip,
			ip_ranges.start_port,
			ip_ranges.end_port,
			ip_ranges.icmp_type,
			ip_ranges.icmp_code
		FROM egress_policies
		LEFT OUTER JOIN apps ON (egress_policies.source_guid = apps.terminal_guid)
		LEFT OUTER JOIN spaces ON (egress_policies.source_guid = spaces.terminal_guid)
		LEFT OUTER JOIN ip_ranges ON (egress_policies.destination_guid = ip_ranges.terminal_guid)
		LEFT OUTER JOIN destination_metadatas ON (egress_policies.destination_guid = destination_metadatas.terminal_guid)
		%s;`, strings.Join(extraClauses, " "))
}

type sqlRows interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
}

func (e *EgressPolicyTable) convertRowsToEgressPolicies(rows sqlRows) ([]EgressPolicy, error) {
	var foundPolicies []EgressPolicy
	defer rows.Close()
	for rows.Next() {
		var egressPolicyGUID, sourceTerminalGUID, appLifecycle, name, description, destinationGUID, sourceAppGUID, sourceSpaceGUID, protocol, startIP, endIP *string
		var startPort, endPort, icmpType, icmpCode int
		err := rows.Scan(
			&egressPolicyGUID,
			&sourceTerminalGUID,
			&appLifecycle,
			&name,
			&description,
			&sourceAppGUID,
			&sourceSpaceGUID,
			&destinationGUID,
			&protocol,
			&startIP,
			&endIP,
			&startPort,
			&endPort,
			&icmpType,
			&icmpCode)
		if err != nil {
			return foundPolicies, err
		}
		foundPolicies = append(foundPolicies, mapRowToEgressPolicy(
			egressPolicyGUID,
			sourceTerminalGUID,
			appLifecycle,
			name,
			description,
			destinationGUID,
			sourceAppGUID,
			sourceSpaceGUID,
			protocol,
			startIP,
			endIP,
			startPort,
			endPort,
			icmpType,
			icmpCode))
	}
	return foundPolicies, nil
}

func mapRowToEgressPolicy(egressPolicyGUID, sourceTerminalGUID, appLifecycle, name, description, destinationGUID,
	sourceAppGUID, sourceSpaceGUID, protocol, startIP, endIP *string,
	startPort, endPort, icmpType, icmpCode int) EgressPolicy {

	var ports []Ports
	if startPort != 0 && endPort != 0 {
		ports = []Ports{
			{
				Start: startPort,
				End:   endPort,
			},
		}
	}

	var source EgressSource

	switch {
	case sourceSpaceGUID != nil:
		source = EgressSource{
			ID:           *sourceSpaceGUID,
			Type:         "space",
			TerminalGUID: *sourceTerminalGUID,
		}
	default:
		source = EgressSource{
			ID:           *sourceAppGUID,
			Type:         "app",
			TerminalGUID: *sourceTerminalGUID,
		}
	}

	return EgressPolicy{
		ID:     *egressPolicyGUID,
		Source: source,
		Destination: EgressDestination{
			GUID:        *destinationGUID,
			Name:        *name,
			Description: *description,
			Protocol:    *protocol,
			Ports:       ports,
			IPRanges: []IPRange{
				{
					Start: *startIP,
					End:   *endIP,
				},
			},
			ICMPType: icmpType,
			ICMPCode: icmpCode,
		},
		AppLifecycle: *appLifecycle,
	}
}
