package legacynet_test

import (
	"bytes"
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
		logger       *bytes.Buffer
	)
	BeforeEach(func() {
		logChainName = "some-chain"
		logger = &bytes.Buffer{}
		converter = &legacynet.NetOutRuleConverter{Logger: logger}
	})
	Describe("Convert", func() {
		Context("when the protocol is TCP or UDP", func() {
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
				ruleSpec := converter.Convert(netOutRule, logChainName, false)
				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "ACCEPT"},
				))
			})

			Context("when globalLogging is set to true", func() {
				It("returns iptables rules that goto the log chain", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, true)
					Expect(ruleSpec).To(Equal([]rules.IPTablesRule{
						rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
							"--dst-range", "1.1.1.1-2.2.2.2",
							"-m", "tcp", "--destination-port", "9000:9999",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
							"--dst-range", "1.1.1.1-2.2.2.2",
							"-m", "tcp", "--destination-port", "1111:2222",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
							"--dst-range", "3.3.3.3-4.4.4.4",
							"-m", "tcp", "--destination-port", "9000:9999",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
							"--dst-range", "3.3.3.3-4.4.4.4",
							"-m", "tcp", "--destination-port", "1111:2222",
							"-g", logChainName},
					}))
				})
			})

			Context("when Log on the netout rule is set to true", func() {
				BeforeEach(func() {
					netOutRule.Log = true
				})
				It("returns iptables rules that goto the log chain", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, true)
					Expect(ruleSpec).To(ConsistOf(
						rules.IPTablesRule{
							"-m", "iprange", "-p", "tcp",
							"--dst-range", "1.1.1.1-2.2.2.2",
							"-m", "tcp", "--destination-port", "9000:9999",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
							"--dst-range", "1.1.1.1-2.2.2.2",
							"-m", "tcp", "--destination-port", "1111:2222",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
							"--dst-range", "3.3.3.3-4.4.4.4",
							"-m", "tcp", "--destination-port", "9000:9999",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
							"--dst-range", "3.3.3.3-4.4.4.4",
							"-m", "tcp", "--destination-port", "1111:2222",
							"-g", logChainName},
					))
				})
			})

			Context("when the netout rule does not specify ports", func() {
				BeforeEach(func() {
					netOutRule.Ports = nil
				})
				It("adds no iptables rules", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, false)
					Expect(ruleSpec).To(BeEmpty())
				})

				It("logs the warning", func() {
					converter.Convert(netOutRule, logChainName, false)
					Expect(logger.String()).To(ContainSubstring("UDP/TCP rule must specify ports"))
				})
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

			It("converts a netout rule to a list of iptables rules", func() {
				ruleSpec := converter.Convert(netOutRule, logChainName, false)
				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"-m", "iprange",
						"-p", "icmp",
						"--dst-range", "3.3.3.3-4.4.4.4", "-m", "icmp", "--icmp-type", "8/0",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange",
						"-p", "icmp",
						"--dst-range", "5.5.5.5-6.6.6.6", "-m", "icmp", "--icmp-type", "8/0",
						"--jump", "ACCEPT"},
				))
			})

			Context("when the globalLogging is set to true", func() {
				It("returns iptables rules that goto the log chain", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, true)
					Expect(ruleSpec).To(ConsistOf(
						rules.IPTablesRule{"-m", "iprange",
							"-p", "icmp",
							"--dst-range", "3.3.3.3-4.4.4.4", "-m", "icmp", "--icmp-type", "8/0",
							"-g", "some-chain"},
						rules.IPTablesRule{"-m", "iprange",
							"-p", "icmp",
							"--dst-range", "5.5.5.5-6.6.6.6", "-m", "icmp", "--icmp-type", "8/0",
							"-g", "some-chain"},
					))
				})
			})

			Context("when Log on the netout rule is set to true", func() {
				BeforeEach(func() {
					netOutRule.Log = true
				})
				It("returns iptables rules that goto the log chain", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, false)
					Expect(ruleSpec).To(ConsistOf(
						rules.IPTablesRule{"-m", "iprange",
							"-p", "icmp",
							"--dst-range", "3.3.3.3-4.4.4.4", "-m", "icmp", "--icmp-type", "8/0",
							"-g", "some-chain"},
						rules.IPTablesRule{"-m", "iprange",
							"-p", "icmp",
							"--dst-range", "5.5.5.5-6.6.6.6", "-m", "icmp", "--icmp-type", "8/0",
							"-g", "some-chain"},
					))
				})
			})

			Context("when the netout rule does not specify ICMP type or code", func() {
				BeforeEach(func() {
					netOutRule.ICMPs = nil
				})
				It("adds no iptables rules", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, false)
					Expect(ruleSpec).To(BeEmpty())
				})
				It("logs the warning", func() {
					converter.Convert(netOutRule, logChainName, false)
					Expect(logger.String()).To(ContainSubstring("ICMP rule must specify ICMP type/code"))
				})
			})

			Context("when the netout rule does not specify ICMP code", func() {
				BeforeEach(func() {
					netOutRule.ICMPs.Code = nil
				})
				It("adds no iptables rules", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, false)
					Expect(ruleSpec).To(BeEmpty())
				})
				It("logs the warning", func() {
					converter.Convert(netOutRule, logChainName, false)
					Expect(logger.String()).To(ContainSubstring("ICMP rule must specify ICMP type/code"))
				})
			})

			Context("when the netout rule specifies ports", func() {
				BeforeEach(func() {
					netOutRule.Ports = []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					}
				})
				It("adds no iptables rules", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, false)
					Expect(ruleSpec).To(BeEmpty())
				})
				It("logs the warning", func() {
					converter.Convert(netOutRule, logChainName, false)
					Expect(logger.String()).To(ContainSubstring("ICMP rule must not specify ports"))
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

			It("converts a netout rule to a list of iptables rules", func() {
				ruleSpec := converter.Convert(netOutRule, logChainName, false)
				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"-m", "iprange",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"--jump", "ACCEPT"},
				))
			})

			Context("when globalLogging is set to true", func() {
				It("returns iptables rules that goto the log chain", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, true)
					Expect(ruleSpec).To(ConsistOf(
						rules.IPTablesRule{"-m", "iprange",
							"--dst-range", "1.1.1.1-2.2.2.2",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange",
							"--dst-range", "3.3.3.3-4.4.4.4",
							"-g", logChainName},
					))
				})
			})

			Context("when Log on the netout rule is set to true", func() {
				BeforeEach(func() {
					netOutRule.Log = true
				})

				It("returns iptables rules that goto the log chain", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, false)
					Expect(ruleSpec).To(ConsistOf(
						rules.IPTablesRule{"-m", "iprange",
							"--dst-range", "1.1.1.1-2.2.2.2",
							"-g", logChainName},
						rules.IPTablesRule{"-m", "iprange",
							"--dst-range", "3.3.3.3-4.4.4.4",
							"-g", logChainName},
					))
				})
			})

			Context("when the netout rule specifies ports", func() {
				BeforeEach(func() {
					netOutRule.Ports = []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					}
				})
				It("adds no iptables rules", func() {
					ruleSpec := converter.Convert(netOutRule, logChainName, true)
					Expect(ruleSpec).To(BeEmpty())
				})
				It("logs the warning", func() {
					converter.Convert(netOutRule, logChainName, false)
					Expect(logger.String()).To(ContainSubstring("Rule for all protocols (TCP/UDP/ICMP) must not specify ports"))
				})
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
				ruleSpec := converter.BulkConvert(netOutRules, logChainName, false)

				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange",
						"--dst-range", "5.5.5.5-6.6.6.6",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange",
						"--dst-range", "7.7.7.7-8.8.8.8",
						"--jump", "ACCEPT"},
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
				ruleSpec := converter.BulkConvert(netOutRules, logChainName, false)

				Expect(ruleSpec).To(ConsistOf(
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "ACCEPT"},
					rules.IPTablesRule{"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "ACCEPT"},
				))
			})
		})

	})
})
