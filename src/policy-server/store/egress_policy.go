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

func (e *EgressPolicyTable) CreateEgressPolicy(tx db.Transaction, sourceTerminalGUID, destinationTerminalGUID string) (string, error) {
	guid := e.Guids.New()

	_, err := tx.Exec(tx.Rebind(`
			INSERT INTO egress_policies (guid, source_guid, destination_guid)
			VALUES (?,?,?)
		`),
		guid,
		sourceTerminalGUID,
		destinationTerminalGUID,
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

	rows, err := tx.Queryx(tx.Rebind(`
	SELECT
		egress_policies.guid,
		egress_policies.source_guid,
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
	WHERE egress_policies.guid IN (`+generateQuestionMarkString(len(guids))+`);`),
		convertToInterfaceSlice(guids)...)
	if err != nil {
		return []EgressPolicy{}, err
	}

	var foundPolicies []EgressPolicy
	defer rows.Close()
	for rows.Next() {
		var egressPolicyGUID, sourceTerminalGUID, name, description, destinationGUID, sourceAppGUID, sourceSpaceGUID, protocol, startIP, endIP *string
		var startPort, endPort, icmpType, icmpCode int

		err = rows.Scan(&egressPolicyGUID, &sourceTerminalGUID, &name, &description, &sourceAppGUID, &sourceSpaceGUID, &destinationGUID, &protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode)
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

		foundPolicies = append(foundPolicies, EgressPolicy{
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
		})
	}

	return foundPolicies, nil
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
	rows, err := e.Conn.Query(`
	SELECT
		egress_policies.guid,
		egress_policies.source_guid,
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
	LEFT OUTER JOIN destination_metadatas ON (egress_policies.destination_guid = destination_metadatas.terminal_guid);`)

	var foundPolicies []EgressPolicy
	if err != nil {
		return foundPolicies, err
	}

	defer rows.Close()
	for rows.Next() {

		var egressPolicyGUID, sourceTerminalGUID, name, description, destinationGUID, sourceAppGUID, sourceSpaceGUID, protocol, startIP, endIP *string
		var startPort, endPort, icmpType, icmpCode int

		err = rows.Scan(&egressPolicyGUID, &sourceTerminalGUID, &name, &description, &sourceAppGUID, &sourceSpaceGUID, &destinationGUID, &protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode)
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

		foundPolicies = append(foundPolicies, EgressPolicy{
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
		})
	}

	return foundPolicies, nil
}

func (e *EgressPolicyTable) GetBySourceGuids(ids []string) ([]EgressPolicy, error) {
	var foundPolicies []EgressPolicy

	interfaceIds := make([]interface{}, len(ids))
	for i, id := range ids {
		interfaceIds[i] = id
	}

	interfaceIds = append(interfaceIds, interfaceIds...)

	questionMarks := make([]string, len(ids))
	for i := range questionMarks {
		questionMarks[i] = "?"
	}

	questionMarksStr := strings.Join(questionMarks, ",")

	query := fmt.Sprintf(`
	SELECT
		apps.app_guid,
		spaces.space_guid,
		ip_ranges.id,
		ip_ranges.protocol,
		ip_ranges.start_ip,
		ip_ranges.end_ip,
		ip_ranges.start_port,
		ip_ranges.end_port,
		ip_ranges.icmp_type,
		ip_ranges.icmp_code
	FROM egress_policies
	LEFT OUTER JOIN apps on (egress_policies.source_guid = apps.terminal_guid)
	LEFT OUTER JOIN spaces on (egress_policies.source_guid = spaces.terminal_guid)
	LEFT OUTER JOIN ip_ranges on (egress_policies.destination_guid = ip_ranges.terminal_guid)
	WHERE apps.app_guid IN (%s) OR spaces.space_guid IN (%s)
	ORDER BY ip_ranges.id;`, questionMarksStr, questionMarksStr)

	rows, err := e.Conn.Query(e.Conn.Rebind(query), interfaceIds...)
	if err != nil {
		return foundPolicies, err
	}

	defer rows.Close()
	for rows.Next() {

		var sourceAppGUID, sourceSpaceGUID, destinationGUID, protocol, startIP, endIP *string
		var startPort, endPort, icmpType, icmpCode int

		err = rows.Scan(&sourceAppGUID, &sourceSpaceGUID, &destinationGUID, &protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode)
		if err != nil {
			return foundPolicies, err
		}

		var ports []Ports
		if startPort != 0 && endPort != 0 {
			ports = []Ports{
				{
					Start: int(startPort),
					End:   int(endPort),
				},
			}
		}

		var source EgressSource

		switch {
		case sourceSpaceGUID != nil:
			source = EgressSource{
				ID:   *sourceSpaceGUID,
				Type: "space",
			}
		default:
			source = EgressSource{
				ID:   *sourceAppGUID,
				Type: "app",
			}
		}

		foundPolicies = append(foundPolicies, EgressPolicy{
			Source: source,
			Destination: EgressDestination{
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
			},
		})
	}

	return foundPolicies, nil
}
