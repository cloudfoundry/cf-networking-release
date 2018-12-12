package store

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/db"
)

type EgressDestinationTable struct{}

func (e *EgressDestinationTable) All(tx db.Transaction) ([]EgressDestination, error) {
	query := egressDestinationsQuery("")
	rows, err := tx.Queryx(query)
	if err != nil {
		return []EgressDestination{}, err
	}
	defer rows.Close()
	return convertRowsToEgressDestinations(rows)
}

func (e *EgressDestinationTable) GetByGUID(tx db.Transaction, guids ...string) ([]EgressDestination, error) {
	query := egressDestinationsQuery(`WHERE ip_ranges.terminal_guid IN (` + generateQuestionMarkString(len(guids)) + `)`)
	rows, err := tx.Queryx(tx.Rebind(query), convertToInterfaceSlice(guids)...)
	if err != nil {
		return []EgressDestination{}, fmt.Errorf("running query: %s", err)
	}
	defer rows.Close()
	return convertRowsToEgressDestinations(rows)
}

func (e *EgressDestinationTable) GetByName(tx db.Transaction, names ...string) ([]EgressDestination, error) {
	query := egressDestinationsQuery("WHERE d_m.name IN (" + generateQuestionMarkString(len(names)) + ")")
	rows, err := tx.Queryx(tx.Rebind(query), convertToInterfaceSlice(names)...)
	if err != nil {
		return []EgressDestination{}, fmt.Errorf("running query: %s", err)
	}
	defer rows.Close()
	return convertRowsToEgressDestinations(rows)
}

func (e *EgressDestinationTable) Delete(tx db.Transaction, guid string) error {
	_, err := tx.Exec(tx.Rebind(`DELETE FROM ip_ranges WHERE terminal_guid = ?`), guid)
	return err
}

func (e *EgressDestinationTable) UpdateIPRange(tx db.Transaction, destinationTerminalGUID, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) error {
	_, err := tx.Exec(tx.Rebind(`
	  UPDATE ip_ranges
		SET protocol=?, start_ip=?, end_ip=?, start_port=?, end_port=?, icmp_type=?, icmp_code=?
		WHERE terminal_guid=?
	`),
		protocol,
		startIP,
		endIP,
		startPort,
		endPort,
		icmpType,
		icmpCode,
		destinationTerminalGUID,
	)

	return err
}

func (e *EgressDestinationTable) CreateIPRange(tx db.Transaction, destinationTerminalGUID, startIP, endIP, protocol string, startPort, endPort, icmpType, icmpCode int64) error {
	_, err := tx.Exec(tx.Rebind(`
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
		return fmt.Errorf("error inserting ip ranges: %s", err)
	}

	return nil
}

func convertRowsToEgressDestinations(rows sqlRows) ([]EgressDestination, error) {
	foundEgressDestinationIndexes := make(map[string]int)
	var destinationsToReturn []EgressDestination
	var count int

	for rows.Next() {
		var (
			startPort, endPort, icmpType, icmpCode                    int
			terminalGUID, name, description, protocol, startIP, endIP *string
			ports                                                     []Ports
		)

		err := rows.Scan(&protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode, &terminalGUID, &name, &description)

		if err != nil {
			return []EgressDestination{}, err
		}

		ports = []Ports{}
		if startPort != 0 && endPort != 0 {
			ports = []Ports{{Start: startPort, End: endPort}}
		}

		if destinationIdx, ok := foundEgressDestinationIndexes[*terminalGUID]; ok {
			destination := destinationsToReturn[destinationIdx]
			destination.Rules = append(destination.Rules, EgressDestinationRule{
				Protocol: *protocol,
				Ports:    ports,
				IPRanges: []IPRange{{Start: *startIP, End: *endIP}},
				ICMPType: icmpType,
				ICMPCode: icmpCode,
			})
			destinationsToReturn[destinationIdx] = destination
		} else {
			destination := EgressDestination{
				GUID:        *terminalGUID,
				Name:        *name,
				Description: *description,
				Rules: []EgressDestinationRule{
					{
						Protocol: *protocol,
						Ports:    ports,
						IPRanges: []IPRange{{Start: *startIP, End: *endIP}},
						ICMPType: icmpType,
						ICMPCode: icmpCode,
					},
				},
			}
			destinationsToReturn = append(destinationsToReturn, destination)
			foundEgressDestinationIndexes[*terminalGUID] = count
			count = count + 1
		}
	}

	return destinationsToReturn, nil
}

func egressDestinationsQuery(whereClause string) string {
	return strings.Join([]string{`SELECT
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
		  ON d_m.terminal_guid = ip_ranges.terminal_guid`,
		whereClause,
		`ORDER BY ip_ranges.id`}, " ")
}

func generateQuestionMarkString(length int) string {
	questionMarks := make([]string, length)
	for i := 0; i < length; i++ {
		questionMarks[i] = "?"
	}
	return strings.Join(questionMarks, ", ")
}

func convertToInterfaceSlice(slice []string) []interface{} {
	ifaces := make([]interface{}, len(slice))
	for i, value := range slice {
		ifaces[i] = value
	}
	return ifaces
}
