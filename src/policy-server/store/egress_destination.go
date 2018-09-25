package store

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

type EgressDestinationTable struct{}

func (e *EgressDestinationTable) GetByGUID(tx db.Transaction, guids ...string) ([]EgressDestination, error) {
	questionMarks := make([]string, len(guids))
	for i := range questionMarks {
		questionMarks[i] = "?"
	}
	questionMarksStr := strings.Join(questionMarks, ", ")
	guidsInterfaces := make([]interface{}, len(guids))
	for i, guid := range guids {
		guidsInterfaces[i] = guid
	}

	rows, err := tx.Queryx(tx.Rebind(`
    SELECT
		ip_ranges.protocol,
		ip_ranges.start_ip,
		ip_ranges.end_ip,
		ip_ranges.start_port,
		ip_ranges.end_port,
		ip_ranges.icmp_type,
		ip_ranges.icmp_code,
		ip_ranges.terminal_guid,
		COALESCE(d_m.name, ''),
		COALESCE(d_m.description, '')
	FROM ip_ranges
	LEFT OUTER JOIN destination_metadatas AS d_m
	  ON d_m.terminal_guid = ip_ranges.terminal_guid
	WHERE ip_ranges.terminal_guid IN (`+questionMarksStr+`)
	ORDER BY ip_ranges.id
	`), guidsInterfaces...)

	if err != nil {
		return []EgressDestination{}, fmt.Errorf("running query: %s", err)
	}

	var egressDestinations []EgressDestination
	for rows.Next() {
		var (
			startPort, endPort, icmpType, icmpCode                    int
			terminalGUID, name, description, protocol, startIP, endIP *string
			ports                                                     []Ports
		)

		err = rows.Scan(&protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode, &terminalGUID, &name, &description)
		if err != nil {
			return []EgressDestination{}, fmt.Errorf("scanning row: %s", err)
		}

		if startPort != 0 && endPort != 0 {
			ports = []Ports{{Start: startPort, End: endPort}}
		}

		egressDestinations = append(egressDestinations, EgressDestination{
			GUID:        *terminalGUID,
			Name:        *name,
			Description: *description,
			Protocol:    *protocol,
			Ports:       ports,
			IPRanges:    []IPRange{{Start: *startIP, End: *endIP}},
			ICMPType:    icmpType,
			ICMPCode:    icmpCode,
		})
	}

	return egressDestinations, nil
}

func (e *EgressDestinationTable) CreateIPRange(tx db.Transaction, destinationTerminalGUID, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) (int64, error) {
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

func (e *EgressDestinationTable) Delete(tx db.Transaction, guid string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM ip_ranges WHERE terminal_guid = ?`), guid)
	return err
}

func (e *EgressDestinationTable) All(tx db.Transaction) ([]EgressDestination, error) {
	rows, err := tx.Queryx(`
    SELECT
		ip_ranges.protocol,
		ip_ranges.start_ip,
		ip_ranges.end_ip,
		ip_ranges.start_port,
		ip_ranges.end_port,
		ip_ranges.icmp_type,
		ip_ranges.icmp_code,
		ip_ranges.terminal_guid,
		COALESCE(d_m.name, ''),
		COALESCE(d_m.description, '')
	FROM ip_ranges
	LEFT OUTER JOIN destination_metadatas AS d_m
	  ON d_m.terminal_guid = ip_ranges.terminal_guid
	ORDER BY ip_ranges.id;`)
	if err != nil {
		return []EgressDestination{}, err
	}
	defer rows.Close()

	var foundEgressDestinations []EgressDestination

	for rows.Next() {
		var (
			startPort, endPort, icmpType, icmpCode                    int
			terminalGUID, name, description, protocol, startIP, endIP *string
			ports                                                     []Ports
		)

		err = rows.Scan(&protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode, &terminalGUID, &name, &description)

		if err != nil {
			return []EgressDestination{}, err
		}

		if startPort != 0 && endPort != 0 {
			ports = []Ports{{Start: startPort, End: endPort}}
		}

		foundEgressDestinations = append(foundEgressDestinations, EgressDestination{
			GUID:        *terminalGUID,
			Name:        *name,
			Description: *description,
			Protocol:    *protocol,
			Ports:       ports,
			IPRanges:    []IPRange{{Start: *startIP, End: *endIP}},
			ICMPType:    icmpType,
			ICMPCode:    icmpCode,
		})
	}
	return foundEgressDestinations, nil
}
