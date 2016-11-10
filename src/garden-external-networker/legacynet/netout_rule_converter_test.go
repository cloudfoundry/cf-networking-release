package legacynet_test

import (
	"garden-external-networker/legacynet"
	"lib/rules"
	"net"

	"code.cloudfoundry.org/garden"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetOutRuleConverter", func() {
	var (
		converter  *legacynet.NetOutRuleConverter
		netOutRule garden.NetOutRule
	)
	BeforeEach(func() {
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
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4")

				Expect(ruleSpec).To(ConsistOf(
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"}},
				))
			})
		})

		Context("when ports or protocol are not specified", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
				}
			})
			It("returns IP tables rules without ports or protocol", func() {
				ruleSpec := converter.Convert(netOutRule, "1.2.3.4")
				Expect(ruleSpec).To(ConsistOf(
					rules.GenericRule{[]string{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"--jump", "RETURN"}},
				))
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
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("5.5.5.5"), End: net.ParseIP("6.6.6.6")},
						{Start: net.ParseIP("7.7.7.7"), End: net.ParseIP("8.8.8.8")},
					},
				}
				netOutRules = []garden.NetOutRule{rule1, rule2}
			})

			It("converts a netout rule to a list of iptables rules", func() {
				ruleSpec := converter.BulkConvert(netOutRules, "1.2.3.4")

				Expect(ruleSpec).To(ConsistOf(
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange",
						"--dst-range", "5.5.5.5-6.6.6.6",
						"--jump", "RETURN"}},
					rules.GenericRule{[]string{"--source", "1.2.3.4",
						"-m", "iprange",
						"--dst-range", "7.7.7.7-8.8.8.8",
						"--jump", "RETURN"}},
				))
			})
		})

	})
})
