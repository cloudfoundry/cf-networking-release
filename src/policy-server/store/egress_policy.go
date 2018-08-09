package store

import (
	"database/sql"
	"fmt"
	"policy-server/db"
	"strings"
)

type EgressPolicyTable struct {
	Conn Database
}

func (e *EgressPolicyTable) CreateTerminal(tx db.Transaction) (int64, error) {
	driverName := tx.DriverName()

	if driverName == "mysql" {
		result, err := tx.Exec("INSERT INTO terminals (id) VALUES (NULL)")
		if err != nil {
			return -1, err
		}

		return result.LastInsertId()

	} else if driverName == "postgres" {
		var id int64
		err := tx.QueryRow("INSERT INTO terminals default values RETURNING id").Scan(&id)
		if err != nil {
			return -1, err
		}

		return id, nil
	}

	return -1, fmt.Errorf("unknown driver: %s", driverName)
}

func (e *EgressPolicyTable) CreateApp(tx db.Transaction, sourceTerminalID int64, appGUID string) (int64, error) {
	driverName := tx.DriverName()

	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO apps (terminal_id, app_guid)
			VALUES (?,?)
		`),
			sourceTerminalID,
			appGUID,
		)
		if err != nil {
			return -1, err
		}

		return result.LastInsertId()
	} else if driverName == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO apps (terminal_id, app_guid)
			VALUES (?,?)
			RETURNING id
		`),
			sourceTerminalID,
			appGUID,
		).Scan(&id)

		if err != nil {
			return -1, fmt.Errorf("error inserting app: %s", err)
		}

		return id, nil
	}
	return -1, fmt.Errorf("unknown driver: %s", driverName)
}

func (e *EgressPolicyTable) CreateIPRange(tx db.Transaction, destinationTerminalID int64, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) (int64, error) {
	driverName := tx.DriverName()
	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id, start_port, end_port, icmp_type, icmp_code)
			VALUES (?,?,?,?,?,?,?,?)
		`),
			protocol,
			startIP,
			endIP,
			destinationTerminalID,
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
			INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id, start_port, end_port, icmp_type, icmp_code)
			VALUES (?,?,?,?,?,?,?,?)
			RETURNING id
		`),
			protocol,
			startIP,
			endIP,
			destinationTerminalID,
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

func (e *EgressPolicyTable) CreateEgressPolicy(tx db.Transaction, sourceTerminalID, destinationTerminalID int64) (int64, error) {
	driverName := tx.DriverName()
	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO egress_policies (source_id, destination_id)
			VALUES (?,?)
		`),
			sourceTerminalID,
			destinationTerminalID,
		)

		if err != nil {
			return -1, fmt.Errorf("error inserting egress policy: %s", err)
		}

		return result.LastInsertId()
	} else if driverName == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO egress_policies (source_id, destination_id)
			VALUES (?,?)
			RETURNING id
		`),
			sourceTerminalID,
			destinationTerminalID,
		).Scan(&id)

		if err != nil {
			return -1, fmt.Errorf("error inserting egress policy: %s", err)
		}

		return id, nil
	}

	return -1, fmt.Errorf("unknown driver: %s", driverName)
}

func (e *EgressPolicyTable) CreateSpace(tx db.Transaction, sourceTerminalID int64, spaceGUID string) (int64, error) {
	driverName := tx.DriverName()

	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO spaces (terminal_id, space_guid)
			VALUES (?,?)
		`),
			sourceTerminalID,
			spaceGUID,
		)
		if err != nil {
			return -1, err
		}

		return result.LastInsertId()
	} else if driverName == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO spaces (terminal_id, space_guid)
			VALUES (?,?)
			RETURNING id
		`),
			sourceTerminalID,
			spaceGUID,
		).Scan(&id)

		if err != nil {
			return -1, fmt.Errorf("error inserting space: %s", err)
		}

		return id, nil
	}
	return -1, fmt.Errorf("unknown driver: %s", driverName)
}

func (e *EgressPolicyTable) DeleteEgressPolicy(tx db.Transaction, egressPolicyID int64) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM egress_policies WHERE id = ?`), egressPolicyID)
	return err
}

func (e *EgressPolicyTable) DeleteIPRange(tx db.Transaction, ipRangeID int64) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM ip_ranges WHERE id = ?`), ipRangeID)
	return err
}

func (e *EgressPolicyTable) DeleteTerminal(tx db.Transaction, terminalID int64) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM terminals WHERE id = ?`), terminalID)
	return err
}

func (e *EgressPolicyTable) DeleteApp(tx db.Transaction, appID int64) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM apps WHERE id = ?`), appID)
	return err
}

func (e *EgressPolicyTable) DeleteSpace(tx db.Transaction, spaceID int64) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM spaces WHERE id = ?`), spaceID)
	return err
}

func (e *EgressPolicyTable) IsTerminalInUse(tx db.Transaction, terminalID int64) (bool, error) {
	var count int64
	err := tx.QueryRow(tx.Rebind(`SELECT COUNT(id) FROM egress_policies WHERE source_id = ? OR destination_id = ?`), terminalID, terminalID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (e *EgressPolicyTable) GetIDsByEgressPolicy(tx db.Transaction, egressPolicy EgressPolicy) (EgressPolicyIDCollection, error) {
	var egressPolicyID, sourceTerminalID, destinationTerminalID, sourceID, appID, spaceID, ipRangeID int64

	var startPort, endPort int64
	if len(egressPolicy.Destination.Ports) > 0 {
		startPort = int64(egressPolicy.Destination.Ports[0].Start)
		endPort = int64(egressPolicy.Destination.Ports[0].End)
	}

	var sourceTable, sourceGUIDColumn string
	switch egressPolicy.Source.Type {
	case "space":
		sourceTable = "spaces"
		sourceGUIDColumn = "space_guid"
	default:
		sourceTable = "apps"
		sourceGUIDColumn = "app_guid"
	}

	err := tx.QueryRow(tx.Rebind(fmt.Sprintf(`
		SELECT
			egress_policies.id,
			egress_policies.source_id,
			egress_policies.destination_id,
			%s.id,
			ip_ranges.id
		FROM egress_policies
		JOIN %[1]s on (egress_policies.source_id = %[1]s.terminal_id)
		JOIN ip_ranges on (egress_policies.destination_id = ip_ranges.terminal_id)
		WHERE %[1]s.%[2]s = ? AND
			ip_ranges.protocol = ? AND
			ip_ranges.start_ip = ? AND
			ip_ranges.end_ip = ? AND
			ip_ranges.start_port = ? AND
			ip_ranges.end_port = ? AND
			ip_ranges.icmp_type = ? AND
			ip_ranges.icmp_code = ?
		;`, sourceTable, sourceGUIDColumn)),
		egressPolicy.Source.ID,
		egressPolicy.Destination.Protocol,
		egressPolicy.Destination.IPRanges[0].Start,
		egressPolicy.Destination.IPRanges[0].End,
		startPort,
		endPort,
		egressPolicy.Destination.ICMPType,
		egressPolicy.Destination.ICMPCode,
	).Scan(&egressPolicyID, &sourceTerminalID, &destinationTerminalID, &sourceID, &ipRangeID)
	if err != nil {
		return EgressPolicyIDCollection{}, err
	}

	switch egressPolicy.Source.Type {
	case "space":
		appID = -1
		spaceID = sourceID
	default:
		spaceID = -1
		appID = sourceID
	}

	policyIDs := EgressPolicyIDCollection{
		EgressPolicyID:        egressPolicyID,
		DestinationTerminalID: destinationTerminalID,
		DestinationIPRangeID:  ipRangeID,
		SourceTerminalID:      sourceTerminalID,
		SourceAppID:           appID,
		SourceSpaceID:         spaceID,
	}

	return policyIDs, nil
}

func (e *EgressPolicyTable) GetTerminalByAppGUID(tx db.Transaction, appGUID string) (int64, error) {
	var id int64

	err := tx.QueryRow(tx.Rebind(`
	SELECT terminal_id FROM apps WHERE app_guid = ?
	`),
		appGUID,
	).Scan(&id)

	if err != nil && err == sql.ErrNoRows {
		return -1, nil
	} else {
		return id, err
	}
}

func (e *EgressPolicyTable) GetTerminalBySpaceGUID(tx db.Transaction, spaceGUID string) (int64, error) {
	var id int64

	err := tx.QueryRow(tx.Rebind(`
	SELECT terminal_id FROM spaces WHERE space_guid = ?
	`),
		spaceGUID,
	).Scan(&id)

	if err != nil && err == sql.ErrNoRows {
		return -1, nil
	} else {
		return id, err
	}
}

func (e *EgressPolicyTable) GetAllPolicies() ([]EgressPolicy, error) {
	rows, err := e.Conn.Query(`
	SELECT
		apps.app_guid,
		spaces.space_guid,
		ip_ranges.protocol,
		ip_ranges.start_ip,
		ip_ranges.end_ip,
		ip_ranges.start_port,
		ip_ranges.end_port,
		ip_ranges.icmp_type,
		ip_ranges.icmp_code
	FROM egress_policies
	LEFT OUTER JOIN apps on (egress_policies.source_id = apps.terminal_id)
	LEFT OUTER JOIN spaces on (egress_policies.source_id = spaces.terminal_id)
	LEFT OUTER JOIN ip_ranges on (egress_policies.destination_id = ip_ranges.terminal_id);`)

	var foundPolicies []EgressPolicy
	if err != nil {
		return foundPolicies, err
	}

	defer rows.Close()
	for rows.Next() {

		var sourceAppGUID, sourceSpaceGUID, protocol, startIP, endIP *string
		var startPort, endPort, icmpType, icmpCode int

		err = rows.Scan(&sourceAppGUID, &sourceSpaceGUID, &protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode)
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

func (e *EgressPolicyTable) GetByGuids(ids []string) ([]EgressPolicy, error) {
	foundPolicies := []EgressPolicy{}

	for i, id := range ids {
		ids[i] = fmt.Sprintf("'%s'", id)
	}

	query := fmt.Sprintf(`
	SELECT
		apps.app_guid,
		spaces.space_guid,
		ip_ranges.protocol,
		ip_ranges.start_ip,
		ip_ranges.end_ip,
		ip_ranges.start_port,
		ip_ranges.end_port,
		ip_ranges.icmp_type,
		ip_ranges.icmp_code
	FROM egress_policies
	LEFT OUTER JOIN apps on (egress_policies.source_id = apps.terminal_id)
	LEFT OUTER JOIN spaces on (egress_policies.source_id = spaces.terminal_id)
	LEFT OUTER JOIN ip_ranges on (egress_policies.destination_id = ip_ranges.terminal_id)
	WHERE apps.app_guid IN (%s) OR spaces.space_guid IN (%s);`, strings.Join(ids, ","), strings.Join(ids, ","))
	rows, err := e.Conn.Query(query)
	if err != nil {
		return foundPolicies, err
	}

	defer rows.Close()
	for rows.Next() {

		var sourceAppGUID, sourceSpaceGUID, protocol, startIP, endIP *string
		var startPort, endPort, icmpType, icmpCode int

		err = rows.Scan(&sourceAppGUID, &sourceSpaceGUID, &protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode)
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
