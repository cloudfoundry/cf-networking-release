package legacynet_test

import (
	"cni-wrapper-plugin/legacynet"
	"lib/rules"
	"net"

	"code.cloudfoundry.org/garden"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetOutRuleConverter", func() {
	var (
		converter    *legacynet.NetOutRuleConverter
		netOutRule   garden.NetOutRule
		logChainName string
	)
	BeforeEach(func() {
		logChainName = "some-chain"
		converter = &legacynet.NetOutRuleConverter{}
	})
	Describe("Convert", func() {
		Context("when ports and protocol are specified", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					},
				}
			})
			It("converts a netout rule to a list of iptables rules", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)

				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
				))
			})
		})

		Context("when Convert is called with globalLogging set to true", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
				}
			})
			It("returns IP tables rules that goto the log chain", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, true)
				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-g", "some-chain"},
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-g", "some-chain"},
				))
			})
		})

		Context("when logging is enabled", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
					Log: true,
				}
			})
			It("returns IP tables rules without ports or protocol", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-g", "some-chain"},
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-g", "some-chain"},
				))
			})
		})

		Context("when the protocol is ICMP", func() {
			BeforeEach(func() {
				var code garden.ICMPCode = 0
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolICMP,
					Networks: []garden.IPRange{
						{
							Start: net.ParseIP("3.3.3.3"),
							End:   net.ParseIP("4.4.4.4"),
						}, {
							Start: net.ParseIP("5.5.5.5"),
							End:   net.ParseIP("6.6.6.6"),
						},
					},
					ICMPs: &garden.ICMPControl{
						Type: 8,
						Code: &code,
					},
				}
			})

			It("returns IP tables rules with ICMP code", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"-p", "icmp",
						"--dst-range", "3.3.3.3-4.4.4.4", "-m", "icmp", "--icmp-type", "8/0",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"-p", "icmp",
						"--dst-range", "5.5.5.5-6.6.6.6", "-m", "icmp", "--icmp-type", "8/0",
						"--jump", "RETURN"},
				))
			})

			Context("when the logging is enabled", func() {
				BeforeEach(func() {
					netOutRule.Log = true
				})
				It("returns IP tables logging rules with ICMP code", func() {
					ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
					Expect(ruleSpec).To(ConsistOf(
						rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
							"-p", "icmp",
							"--dst-range", "3.3.3.3-4.4.4.4", "-m", "icmp", "--icmp-type", "8/0",
							"-g", "some-chain"},
						rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
							"-p", "icmp",
							"--dst-range", "5.5.5.5-6.6.6.6", "-m", "icmp", "--icmp-type", "8/0",
							"-g", "some-chain"},
					))
				})
			})
		})

		Context("when protocol is all", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolAll,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
				}
			})
			It("returns IP tables rules without ports or protocol", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"--jump", "RETURN"},
				))
			})
		})

		Context("when ports are not specified but protocol is udp/tcp", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolUDP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
				}
			})
			It("return no rules", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
				Expect(ruleSpec).To(BeEmpty())
			})
		})

		Context("when ports are specified but protocol is icmp", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolICMP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					},
				}
			})
			It("return no rules", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
				Expect(ruleSpec).To(BeEmpty())
			})
		})

		Context("when type not specified but protocol is icmp", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolICMP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
				}
			})
			It("return no rules", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
				Expect(ruleSpec).To(BeEmpty())
			})
		})

		Context("when ports are specified but protocol is all", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolAll,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					},
				}
			})
			It("return no rules", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4", logChainName, false)
				Expect(ruleSpec).To(BeEmpty())
			})
		})

	})

	Describe("BulkConvert", func() {
		var netOutRules []garden.NetOutRule
		Context("converts multiple net out rules to generic rules", func() {
			BeforeEach(func() {
				rule1 := garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					},
				}
				rule2 := garden.NetOutRule{
					Protocol: garden.ProtocolAll,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("5.5.5.5"), End: net.ParseIP("6.6.6.6")},
						{Start: net.ParseIP("7.7.7.7"), End: net.ParseIP("8.8.8.8")},
					},
				}
				netOutRules = []garden.NetOutRule{rule1, rule2}
			})

			It("converts a netout rule to a list of iptables rules", func() {
				ruleSpec := converter.BulkConvert(netOutRules, "1.2.3.4", logChainName, false)

				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange",
						"--dst-range", "5.5.5.5-6.6.6.6",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange",
						"--dst-range", "7.7.7.7-8.8.8.8",
						"--jump", "RETURN"},
				))
			})
		})

		Context("when a net out rule is invalid", func() {
			BeforeEach(func() {
				rule1 := garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					},
				}
				invalidRule := garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("5.5.5.5"), End: net.ParseIP("6.6.6.6")},
						{Start: net.ParseIP("7.7.7.7"), End: net.ParseIP("8.8.8.8")},
					},
				}
				netOutRules = []garden.NetOutRule{rule1, invalidRule}
			})

			It("does not include iptables rules for invalid netout rules", func() {
				ruleSpec := converter.BulkConvert(netOutRules, "1.2.3.4", logChainName, false)

				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					rules.IPTablesRule{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
				))
			})
		})

	})
})
