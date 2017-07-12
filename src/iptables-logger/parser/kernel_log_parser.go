package parser

import (
	"strconv"
	"strings"
)

type ParsedData struct {
	Direction       string `json:"direction"`
	Allowed         bool   `json:"allowed"`
	SourceIP        string `json:"src_ip"`
	DestinationIP   string `json:"dst_ip"`
	SourcePort      int    `json:"src_port"`
	DestinationPort int    `json:"dst_port"`
	Protocol        string `json:"protocol"`
	Mark            string `json:"mark"`
	ICMPType        int    `json:"icmp_type"`
	ICMPCode        int    `json:"icmp_code"`
}

type KernelLogParser struct {
}

func (k *KernelLogParser) IsIPTablesLogData(line string) bool {
	return strings.Contains(line, "OK_") || strings.Contains(line, "DENY_")
}

func (k *KernelLogParser) Parse(line string) ParsedData {
	if !k.IsIPTablesLogData(line) {
		return ParsedData{}
	}

	data := map[string]string{}
	words := strings.Fields(line)
	for _, word := range words {
		if equalSignIndex := strings.Index(word, "="); equalSignIndex > -1 {
			key := word[:equalSignIndex]
			value := word[equalSignIndex+1:]
			if _, ok := data[key]; !ok {
				data[key] = value
			}
		}
	}

	allowed := strings.Contains(line, "OK_")
	var direction string
	if strings.Contains(data["OUT"], "s-") {
		direction = "ingress"
	} else {
		direction = "egress"
	}

	sourcePort, err := strconv.Atoi(data["SPT"])
	if err != nil {
		sourcePort = 0
	}
	destinationPort, err := strconv.Atoi(data["DPT"])
	if err != nil {
		destinationPort = 0
	}
	icmpType, err := strconv.Atoi(data["TYPE"])
	if err != nil {
		icmpType = 0
	}
	icmpCode, err := strconv.Atoi(data["CODE"])
	if err != nil {
		icmpCode = 0
	}

	parsed := ParsedData{
		Direction:       direction,
		Allowed:         allowed,
		SourceIP:        data["SRC"],
		DestinationIP:   data["DST"],
		SourcePort:      sourcePort,
		DestinationPort: destinationPort,
		Mark:            data["MARK"],
		Protocol:        data["PROTO"],
		ICMPType:        icmpType,
		ICMPCode:        icmpCode,
	}
	return parsed
}
