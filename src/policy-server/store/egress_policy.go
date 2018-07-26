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

func (e *EgressPolicyTable) CreateIPRange(tx db.Transaction, destinationTerminalID int64, startIP, endIP, protocol string) (int64, error) {
	driverName := tx.DriverName()
	if driverName == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id) 
			VALUES (?,?,?,?)
			`),
			protocol,
			startIP,
			endIP,
			destinationTerminalID,
		)

		if err != nil {
			return -1, fmt.Errorf("error inserting ip ranges: %s", err)
		}

		return result.LastInsertId()
	} else if driverName == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id) 
			VALUES (?,?,?,?)
 			RETURNING id
			`),
			protocol,
			startIP,
			endIP,
			destinationTerminalID,
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

func (e *EgressPolicyTable) IsTerminalInUse(tx db.Transaction, terminalID int64) (bool, error) {
	var count int64
	err := tx.QueryRow(tx.Rebind(`SELECT COUNT(id) FROM egress_policies WHERE source_id = ? OR destination_id = ?`), terminalID, terminalID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (e *EgressPolicyTable) GetIDsByEgressPolicy(tx db.Transaction, egressPolicy EgressPolicy) (EgressPolicyIDCollection, error) {
	var egressPolicyID, sourceTerminalID, destinationTerminalID, appID, ipRangeID int64

	err := tx.QueryRow(tx.Rebind(`
		SELECT
			egress_policies.id,
			egress_policies.source_id,
			egress_policies.destination_id,
			apps.id,
			ip_ranges.id
		from egress_policies
		JOIN apps on (egress_policies.source_id = apps.terminal_id)
		JOIN ip_ranges on (egress_policies.destination_id = ip_ranges.terminal_id)
		WHERE apps.app_guid = ? AND
		      ip_ranges.protocol = ? AND
					ip_ranges.start_ip = ? AND
					ip_ranges.end_ip = ?
		;`),
		egressPolicy.Source.ID,
		egressPolicy.Destination.Protocol,
		egressPolicy.Destination.IPRanges[0].Start,
		egressPolicy.Destination.IPRanges[0].End).
		Scan(&egressPolicyID, &sourceTerminalID, &destinationTerminalID, &appID, &ipRangeID)

	var policyIDs EgressPolicyIDCollection
	if err != nil {
		return policyIDs, err
	}

	policyIDs = EgressPolicyIDCollection{
		EgressPolicyID:        egressPolicyID,
		DestinationTerminalID: destinationTerminalID,
		DestinationIPRangeID:  ipRangeID,
		SourceTerminalID:      sourceTerminalID,
		SourceAppID:           appID,
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

func (e *EgressPolicyTable) GetAllPolicies() ([]EgressPolicy, error) {
	rows, err := e.Conn.Query(`
		SELECT
			apps.app_guid,
			ip_ranges.protocol,
			ip_ranges.start_ip,
			ip_ranges.end_ip
		from egress_policies
		LEFT OUTER JOIN apps on (egress_policies.source_id = apps.terminal_id)
		LEFT OUTER JOIN ip_ranges on (egress_policies.destination_id = ip_ranges.terminal_id);`)

	var foundPolicies []EgressPolicy
	if err != nil {
		return foundPolicies, err
	}

	defer rows.Close()
	for rows.Next() {

		var sourceAppGUID, protocol, startIP, endIP string

		err = rows.Scan(&sourceAppGUID, &protocol, &startIP, &endIP)
		if err != nil {
			return []EgressPolicy{}, err
		}

		foundPolicies = append(foundPolicies, EgressPolicy{
			Source: EgressSource{
				ID: sourceAppGUID,
			},
			Destination: EgressDestination{
				Protocol: protocol,
				IPRanges: []IPRange{
					{
						Start: startIP,
						End:   endIP,
					},
				},
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
			ip_ranges.protocol,
			ip_ranges.start_ip,
			ip_ranges.end_ip
		from egress_policies
		LEFT OUTER JOIN apps on (egress_policies.source_id = apps.terminal_id)
		LEFT OUTER JOIN ip_ranges on (egress_policies.destination_id = ip_ranges.terminal_id)
		WHERE apps.app_guid IN (%s);`, strings.Join(ids, ","))
	rows, err := e.Conn.Query(query)
	if err != nil {
		return foundPolicies, err
	}

	defer rows.Close()
	for rows.Next() {

		var sourceAppGUID, protocol, startIP, endIP string

		err = rows.Scan(&sourceAppGUID, &protocol, &startIP, &endIP)
		if err != nil {
			return foundPolicies, err
		}

		foundPolicies = append(foundPolicies, EgressPolicy{
			Source: EgressSource{
				ID: sourceAppGUID,
			},
			Destination: EgressDestination{
				Protocol: protocol,
				IPRanges: []IPRange{
					{
						Start: startIP,
						End:   endIP,
					},
				},
			},
		})
	}

	return foundPolicies, nil
}
