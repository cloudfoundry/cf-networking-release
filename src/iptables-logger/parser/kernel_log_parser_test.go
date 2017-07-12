package parser_test

import (
	"iptables-logger/parser"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	ingressDeniedTCP  = "May  3 23:34:07 localhost kernel: [87921.493829] DENY_C2C_cb40f81e-52ce-41c5- IN=s-010255015007 OUT=s-010255015013 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=10.255.15.13 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=35889 DF PROTO=TCP SPT=36004 DPT=723 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0x2"
	ingressAllowedTCP = "May  3 23:35:07 localhost kernel: [87981.320056] OK_0002_e9e8959f-3828-4136-8 IN=s-010255015007 OUT=s-010255015013 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=10.255.15.13 LEN=52 TOS=0x00 PREC=0x00 TTL=63 ID=43997 DF PROTO=TCP SPT=60012 DPT=8080 WINDOW=237 RES=0x00 ACK URGP=0 MARK=0x2"
	egressDeniedTCP   = "May  3 23:35:58 localhost kernel: [88032.025828] DENY_d538d169-f2f6-4587-77b1 IN=s-010255015007 OUT=eth0 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=10.10.10.1 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=61375 DF PROTO=TCP SPT=49466 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0x2"
	egressAllowedTCP  = "May  3 23:35:35 localhost kernel: [88008.920287] OK_d538d169-f2f6-4587-77b1-f IN=s-010255015007 OUT=eth0 MAC=aa:aa:0a:ff:0f:07:ee:ee:0a:ff:0f:07:08:00 SRC=10.255.15.7 DST=173.194.210.139 LEN=60 TOS=0x00 PREC=0x00 TTL=63 ID=45400 DF PROTO=TCP SPT=35236 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0x2"
	egressAllowedUDP  = "Jun 28 18:21:24 localhost kernel: [100471.222018] OK_container-handle-1-longer IN=s-010255178004 OUT=eth0 MAC=aa:aa:0a:ff:b2:04:ee:ee:0a:ff:b2:04:08:00 SRC=10.255.0.1 DST=10.10.10.10 LEN=29 TOS=0x00 PREC=0x00 TTL=63 ID=2806 DF PROTO=UDP SPT=36556 DPT=11111 LEN=9 MARK=0x1"
	egressDeniedICMP  = "May 25 17:19:38 localhost kernel: [173756.041192] DENY_da966cab-6a60-49c4-4f90 IN=s-010247180118 OUT=eth0 MAC=aa:aa:0a:f7:b4:76:ee:ee:0a:f7:b4:76:08:00 SRC=10.247.180.118 DST=10.0.0.1 LEN=84 TOS=0x00 PREC=0x00 TTL=63 ID=58750 DF PROTO=ICMP TYPE=8 CODE=2 ID=172 SEQ=1"
)

var _ = Describe("KernelLogParser", func() {
	var (
		kernelLogParser *parser.KernelLogParser
	)
	BeforeEach(func() {
		kernelLogParser = &parser.KernelLogParser{}

	})
	Describe("IsIPTablesLogData", func() {
		It("returns true if it contains OK_ or DENY_", func() {
			Expect(kernelLogParser.IsIPTablesLogData("stuff OK_ stuff")).To(BeTrue())
			Expect(kernelLogParser.IsIPTablesLogData("stuff DENY_ stuff")).To(BeTrue())
		})

		It("returns false", func() {
			Expect(kernelLogParser.IsIPTablesLogData("stuff stuff")).To(BeFalse())
		})
	})

	Describe("Parsing different kernel log messages for TCP or UDP", func() {
		It("ingress allowed", func() {
			Expect(kernelLogParser.Parse(ingressAllowedTCP)).To(Equal(
				parser.ParsedData{
					Direction:       "ingress",
					Allowed:         true,
					SourceIP:        "10.255.15.7",
					DestinationIP:   "10.255.15.13",
					SourcePort:      60012,
					DestinationPort: 8080,
					Protocol:        "TCP",
					Mark:            "0x2",
					ICMPType:        0,
					ICMPCode:        0,
				},
			))
		})

		It("ingress denied", func() {
			Expect(kernelLogParser.Parse(ingressDeniedTCP)).To(Equal(
				parser.ParsedData{
					Direction:       "ingress",
					Allowed:         false,
					SourceIP:        "10.255.15.7",
					DestinationIP:   "10.255.15.13",
					SourcePort:      36004,
					DestinationPort: 723,
					Protocol:        "TCP",
					Mark:            "0x2",
					ICMPType:        0,
					ICMPCode:        0,
				},
			))
		})

		It("egress allowed", func() {
			Expect(kernelLogParser.Parse(egressAllowedUDP)).To(Equal(
				parser.ParsedData{
					Direction:       "egress",
					Allowed:         true,
					SourceIP:        "10.255.0.1",
					DestinationIP:   "10.10.10.10",
					SourcePort:      36556,
					DestinationPort: 11111,
					Protocol:        "UDP",
					Mark:            "0x1",
					ICMPType:        0,
					ICMPCode:        0,
				},
			))
		})

		It("egress denied", func() {
			Expect(kernelLogParser.Parse(egressDeniedTCP)).To(Equal(
				parser.ParsedData{
					Direction:       "egress",
					Allowed:         false,
					SourceIP:        "10.255.15.7",
					DestinationIP:   "10.10.10.1",
					SourcePort:      49466,
					DestinationPort: 80,
					Protocol:        "TCP",
					Mark:            "0x2",
					ICMPType:        0,
					ICMPCode:        0,
				},
			))
		})
		Describe("Parsing log messages for ICMP", func() {
			It("egress denied", func() {
				Expect(kernelLogParser.Parse(egressDeniedICMP)).To(Equal(
					parser.ParsedData{
						Direction:       "egress",
						Allowed:         false,
						SourceIP:        "10.247.180.118",
						DestinationIP:   "10.0.0.1",
						SourcePort:      0,
						DestinationPort: 0,
						Protocol:        "ICMP",
						Mark:            "",
						ICMPType:        8,
						ICMPCode:        2,
					},
				))
			})
		})
		Context("when there is no parseable data", func() {
			It("returns empty parsed data", func() {
				Expect(kernelLogParser.Parse("stuff")).To(Equal(parser.ParsedData{}))
			})
		})
	})
})
