package rules

import (
	"fmt"
	"strconv"
	"strings"
)

type IPTablesRule []string

func AppendComment(rule IPTablesRule, comment string) IPTablesRule {
	comment = strings.Replace(comment, " ", "_", -1)
	return IPTablesRule(
		append(rule, "-m", "comment", "--comment", comment),
	)
}

func NewPortForwardingRule(hostPort, containerPort int, hostIP, containerIP string) IPTablesRule {
	return IPTablesRule{
		"-d", hostIP, "-p", "tcp",
		"-m", "tcp", "--dport", fmt.Sprintf("%d", hostPort),
		"--jump", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", containerIP, containerPort),
	}
}

func NewIngressMarkRule(hostInterface string, hostPort int, hostIP, tag string) IPTablesRule {
	return IPTablesRule{
		"-i", hostInterface, "-d", hostIP, "-p", "tcp",
		"-m", "tcp", "--dport", fmt.Sprintf("%d", hostPort),
		"--jump", "MARK",
		"--set-mark", fmt.Sprintf("0x%s", tag),
	}
}

func NewMarkAllowRule(destinationIP, protocol string, port int, tag string, sourceAppGUID, destinationAppGUID string) IPTablesRule {
	return AppendComment(IPTablesRule{
		"-d", destinationIP,
		"-p", protocol,
		"--dport", strconv.Itoa(port),
		"-m", "mark", "--mark", fmt.Sprintf("0x%s", tag),
		"--jump", "ACCEPT",
	}, fmt.Sprintf("src:%s_dst:%s", sourceAppGUID, destinationAppGUID))
}

func NewMarkLogRule(destinationIP, protocol string, port int, tag string, destinationAppGUID string) IPTablesRule {
	return IPTablesRule{
		"-d", destinationIP,
		"-p", protocol,
		"--dport", strconv.Itoa(port),
		"-m", "mark", "--mark", fmt.Sprintf("0x%s", tag),
		"-m", "limit", "--limit", "2/min",
		"--jump", "LOG", "--log-prefix",
		trimAndPad(fmt.Sprintf("OK_%s_%s", tag, destinationAppGUID))}
}

func NewMarkSetRule(sourceIP, tag, appGUID string) IPTablesRule {
	return AppendComment(IPTablesRule{
		"--source", sourceIP,
		"--jump", "MARK", "--set-xmark", fmt.Sprintf("0x%s", tag),
	}, fmt.Sprintf("src:%s", appGUID))
}

func NewDefaultEgressRule(localSubnet, deviceName string) IPTablesRule {
	return IPTablesRule{
		"--source", localSubnet,
		"!", "-o", deviceName,
		"--jump", "MASQUERADE",
	}
}

func NewLogRule(rule IPTablesRule, name string) IPTablesRule {
	return IPTablesRule(append(
		rule, "-m", "limit", "--limit", "2/min",
		"--jump", "LOG",
		"--log-prefix", trimAndPad(name),
	))
}

func NewAcceptExistingLocalRule() IPTablesRule {
	return IPTablesRule{
		"-m", "state", "--state", "ESTABLISHED,RELATED",
		"--jump", "ACCEPT",
	}
}

func NewLogLocalRejectRule(localSubnet string) IPTablesRule {
	return NewLogRule(
		IPTablesRule{
			"-s", localSubnet,
			"-d", localSubnet,
		},
		"REJECT_LOCAL: ",
	)
}

func NewDefaultDenyLocalRule(localSubnet string) IPTablesRule {
	return IPTablesRule{
		"--source", localSubnet,
		"-d", localSubnet,
		"--jump", "REJECT",
	}
}

func NewNetOutRule(startIP, endIP string) IPTablesRule {
	return IPTablesRule{
		"-m", "iprange",
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"--jump", "ACCEPT",
	}
}

func NewNetOutWithPortsRule(startIP, endIP string, startPort, endPort int, protocol string) IPTablesRule {
	return IPTablesRule{
		"-m", "iprange",
		"-p", protocol,
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-m", protocol,
		"--destination-port", fmt.Sprintf("%d:%d", startPort, endPort),
		"--jump", "ACCEPT",
	}
}

func NewNetOutICMPRule(startIP, endIP string, icmpType, icmpCode int) IPTablesRule {
	return IPTablesRule{
		"-m", "iprange",
		"-p", "icmp",
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-m", "icmp",
		"--icmp-type", fmt.Sprintf("%d/%d", icmpType, icmpCode),
		"--jump", "ACCEPT",
	}
}

func NewNetOutICMPLogRule(startIP, endIP string, icmpType, icmpCode int, chain string) IPTablesRule {
	return IPTablesRule{
		"-m", "iprange",
		"-p", "icmp",
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-m", "icmp",
		"--icmp-type", fmt.Sprintf("%d/%d", icmpType, icmpCode),
		"-g", chain,
	}
}

func NewNetOutLogRule(startIP, endIP, chain string) IPTablesRule {
	return IPTablesRule{
		"-m", "iprange",
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-g", chain,
	}
}

func NewNetOutWithPortsLogRule(startIP, endIP string, startPort, endPort int, protocol, chain string) IPTablesRule {
	return IPTablesRule{
		"-m", "iprange",
		"-p", protocol,
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-m", protocol,
		"--destination-port", fmt.Sprintf("%d:%d", startPort, endPort),
		"-g", chain,
	}
}

func NewNetOutDefaultLogRule(prefix string) IPTablesRule {
	// TODO make this work for udp
	return IPTablesRule{
		"-p", "tcp",
		"-m", "conntrack", "--ctstate", "INVALID,NEW,UNTRACKED",
		"-j", "LOG", "--log-prefix", trimAndPad(fmt.Sprintf("OK_%s", prefix)),
	}
}

func NewAcceptRule() IPTablesRule {
	return IPTablesRule{
		"--jump", "ACCEPT",
	}
}

func NewInputRelatedEstablishedRule() IPTablesRule {
	return IPTablesRule{
		"-m", "state", "--state", "RELATED,ESTABLISHED",
		"--jump", "ACCEPT",
	}
}

func NewInputAllowRule(protocol, destination string, destPort int) IPTablesRule {
	return IPTablesRule{
		"-p", protocol,
		"-d", destination, "--destination-port", strconv.Itoa(destPort),
		"--jump", "ACCEPT",
	}
}

func NewInputDefaultRejectRule() IPTablesRule {
	return IPTablesRule{
		"--jump", "REJECT",
		"--reject-with", "icmp-port-unreachable",
	}
}

func NewNetOutRelatedEstablishedRule() IPTablesRule {
	return IPTablesRule{
		"-m", "state", "--state", "RELATED,ESTABLISHED",
		"--jump", "ACCEPT",
	}
}

func NewOverlayTagAcceptRule(containerIP, tag string) IPTablesRule {
	return IPTablesRule{
		"-d", containerIP,
		"-m", "mark", "--mark", fmt.Sprintf("0x%s", tag),
		"--jump", "ACCEPT",
	}
}

func NewOverlayDefaultRejectRule(containerIP string) IPTablesRule {
	return IPTablesRule{
		"-d", containerIP,
		"--jump", "REJECT",
		"--reject-with", "icmp-port-unreachable",
	}
}

func NewOverlayDefaultRejectLogRule(containerHandle, containerIP string) IPTablesRule {
	return IPTablesRule{
		"-d", containerIP,
		"-m", "limit", "--limit", "2/min",
		"--jump", "LOG",
		"--log-prefix", trimAndPad(fmt.Sprintf("DENY_C2C_%s", containerHandle)),
	}
}

func NewOverlayAllowEgress(deviceName, containerIP string) IPTablesRule {
	return IPTablesRule{
		"-s", containerIP,
		"-o", deviceName,
		"-m", "mark", "!", "--mark", "0x0",
		"--jump", "ACCEPT",
	}
}

func NewOverlayRelatedEstablishedRule(containerIP string) IPTablesRule {
	return IPTablesRule{
		"-d", containerIP,
		"-m", "state", "--state", "RELATED,ESTABLISHED",
		"--jump", "ACCEPT",
	}
}

func NewNetOutDefaultRejectLogRule(containerHandle string) IPTablesRule {
	return IPTablesRule{
		"-m", "limit", "--limit", "2/min",
		"--jump", "LOG",
		"--log-prefix", trimAndPad(fmt.Sprintf("DENY_%s", containerHandle)),
	}
}

func NewNetOutDefaultRejectRule() IPTablesRule {
	return IPTablesRule{
		"--jump", "REJECT",
		"--reject-with", "icmp-port-unreachable",
	}
}

func trimAndPad(name string) string {
	if len(name) > 28 {
		name = name[:28]
	}
	return fmt.Sprintf(`"%s "`, name)
}
