package store

import (
	"policy-server/db"
	"strconv"
)

type EgressDestinationTable struct{}

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
		ip_ranges.terminal_id
	FROM ip_ranges;`)
	if err != nil {
		return []EgressDestination{}, err
	}
	defer rows.Close()

	var foundEgressDestinations []EgressDestination

	for rows.Next() {
		var (
			terminalID                             int64
			startPort, endPort, icmpType, icmpCode int
			protocol, startIP, endIP               *string
			ports                                  []Ports
		)

		err = rows.Scan(&protocol, &startIP, &endIP, &startPort, &endPort, &icmpType, &icmpCode, &terminalID)

		if err != nil {
			return []EgressDestination{}, err
		}

		if startPort != 0 && endPort != 0 {
			ports = []Ports{{Start: startPort, End: endPort}}
		}

		foundEgressDestinations = append(foundEgressDestinations, EgressDestination{
			ID:          strconv.FormatInt(terminalID, 10),
			Name:        " ",
			Description: " ",
			Protocol:    *protocol,
			Ports:       ports,
			IPRanges:    []IPRange{{Start: *startIP, End: *endIP}},
			ICMPType:    icmpType,
			ICMPCode:    icmpCode,
		})
	}
	return foundEgressDestinations, nil
}
