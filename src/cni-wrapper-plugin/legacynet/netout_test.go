package legacynet_test

import (
	"cni-wrapper-plugin/fakes"
	"cni-wrapper-plugin/legacynet"
	"errors"
	"net"

	"code.cloudfoundry.org/garden"

	lib_fakes "lib/fakes"
	"lib/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Netout", func() {
	var (
		netOut     *legacynet.NetOut
		converter  *fakes.NetOutRuleConverter
		chainNamer *fakes.ChainNamer
		ipTables   *lib_fakes.IPTablesAdapter
	)
	BeforeEach(func() {
		chainNamer = &fakes.ChainNamer{}
		converter = &fakes.NetOutRuleConverter{}
		ipTables = &lib_fakes.IPTablesAdapter{}
		netOut = &legacynet.NetOut{
			ChainNamer: chainNamer,
			IPTables:   ipTables,
			Converter:  converter,
			IngressTag: "FEEDBEEF",
		}
		chainNamer.PrefixStub = func(prefix, handle string) string {
			return prefix + "-" + handle
		}
		chainNamer.PostfixReturns("some-other-chain-name", nil)
	})

	Describe("Initialize", func() {
		It("creates the input chain, netout forwarding chain, and the logging chain", func() {
			err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(3))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("input"))
			Expect(handle).To(Equal("some-container-handle"))

			prefix, handle = chainNamer.PrefixArgsForCall(1)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			prefix, handle = chainNamer.PrefixArgsForCall(2)
			Expect(prefix).To(Equal("overlay"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(chainNamer.PostfixCallCount()).To(Equal(1))
			body, suffix := chainNamer.PostfixArgsForCall(0)
			Expect(body).To(Equal("netout-some-container-handle"))
			Expect(suffix).To(Equal("log"))

			Expect(ipTables.NewChainCallCount()).To(Equal(4))
			table, chain := ipTables.NewChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("input-some-container-handle"))
			table, chain = ipTables.NewChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout-some-container-handle"))
			table, chain = ipTables.NewChainArgsForCall(2)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("overlay-some-container-handle"))
			table, chain = ipTables.NewChainArgsForCall(3)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-other-chain-name"))
		})

		It("inserts a jump rule for the new chains", func() {
			err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.BulkInsertCallCount()).To(Equal(3))
			table, chain, position, rulespec := ipTables.BulkInsertArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("INPUT"))
			Expect(position).To(Equal(1))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{{"--jump", "input-some-container-handle"}}))

			table, chain, position, rulespec = ipTables.BulkInsertArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(position).To(Equal(1))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{{"--jump", "netout-some-container-handle"}}))

			table, chain, position, rulespec = ipTables.BulkInsertArgsForCall(2)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(position).To(Equal(1))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{{"--jump", "overlay-some-container-handle"}}))
		})

		It("writes the default netout and logging rules", func() {
			err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.BulkAppendCallCount()).To(Equal(4))

			table, chain, rulespec := ipTables.BulkAppendArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("input-some-container-handle"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{
				{"-s", "5.6.7.8",
					"-m", "state", "--state", "RELATED,ESTABLISHED",
					"--jump", "ACCEPT"},
				{"-s", "5.6.7.8",
					"--jump", "REJECT",
					"--reject-with", "icmp-port-unreachable"},
			}))

			table, chain, rulespec = ipTables.BulkAppendArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout-some-container-handle"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{
				{"-s", "5.6.7.8",
					"!", "-d", "9.9.0.0/16",
					"-m", "state", "--state", "RELATED,ESTABLISHED",
					"--jump", "ACCEPT"},
				{"-s", "5.6.7.8",
					"!", "-d", "9.9.0.0/16",
					"--jump", "REJECT",
					"--reject-with", "icmp-port-unreachable"},
			}))

			table, chain, rulespec = ipTables.BulkAppendArgsForCall(2)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("overlay-some-container-handle"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{
				{"-d", "5.6.7.8",
					"-m", "state", "--state", "RELATED,ESTABLISHED",
					"--jump", "ACCEPT"},
				{"-d", "5.6.7.8",
					"-m", "mark", "--mark", "0xFEEDBEEF",
					"--jump", "ACCEPT"},
				{"-s", "9.9.0.0/16",
					"-d", "5.6.7.8",
					"--jump", "REJECT",
					"--reject-with", "icmp-port-unreachable"},
			}))

			table, chain, rulespec = ipTables.BulkAppendArgsForCall(3)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-other-chain-name"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{
				{"-p", "tcp",
					"-m", "conntrack", "--ctstate", "INVALID,NEW,UNTRACKED",
					"-j", "LOG", "--log-prefix", "OK_some-container-handle"},
				{"--jump", "ACCEPT"},
			}))
		})

		It("writes the logging rules", func() {
			err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when creating a new chain fails", func() {
			BeforeEach(func() {
				ipTables.NewChainReturns(errors.New("potata"))
			})
			It("returns the error", func() {
				err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
				Expect(err).To(MatchError("creating chain: potata"))
			})
		})

		Context("when the chain namer fails", func() {
			BeforeEach(func() {
				chainNamer.PostfixReturns("", errors.New("banana"))
			})
			It("returns the error", func() {
				err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
				Expect(err).To(MatchError("getting chain name: banana"))
			})
		})

		Context("when inserting a new rule fails", func() {
			BeforeEach(func() {
				ipTables.BulkInsertReturns(errors.New("potato"))
			})
			It("returns the error", func() {
				err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
				Expect(err).To(MatchError("inserting rule: potato"))
			})
		})

		Context("when writing the netout rule fails", func() {
			BeforeEach(func() {
				ipTables.BulkAppendReturns(errors.New("potato"))
			})
			It("returns the error", func() {
				err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
				Expect(err).To(MatchError("appending rule: potato"))
			})
		})

		Context("when global ASG logging is enabled", func() {
			BeforeEach(func() {
				netOut.ASGLogging = true
			})
			It("writes a log rule for denies", func() {
				err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(ipTables.BulkAppendCallCount()).To(Equal(4))

				table, chain, rulespec := ipTables.BulkAppendArgsForCall(1)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("netout-some-container-handle"))
				Expect(rulespec).To(Equal([]rules.IPTablesRule{
					{"-s", "5.6.7.8",
						"!", "-d", "9.9.0.0/16",
						"-m", "state", "--state", "RELATED,ESTABLISHED",
						"--jump", "ACCEPT"},
					{"-s", "5.6.7.8",
						"!", "-d", "9.9.0.0/16",
						"-m", "limit", "--limit", "2/min",
						"--jump", "LOG", "--log-prefix", "DENY_some-container-handle"},
					{"-s", "5.6.7.8",
						"!", "-d", "9.9.0.0/16",
						"--jump", "REJECT",
						"--reject-with", "icmp-port-unreachable"},
				}))
			})
		})

		Context("when C2C logging is enabled", func() {
			BeforeEach(func() {
				netOut.C2CLogging = true
			})
			It("writes a log rule for denies", func() {
				err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(ipTables.BulkAppendCallCount()).To(Equal(4))

				table, chain, rulespec := ipTables.BulkAppendArgsForCall(2)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("overlay-some-container-handle"))
				Expect(rulespec).To(Equal([]rules.IPTablesRule{
					{"-d", "5.6.7.8",
						"-m", "state", "--state", "RELATED,ESTABLISHED",
						"--jump", "ACCEPT"},
					{"-d", "5.6.7.8",
						"-m", "mark", "--mark", "0xFEEDBEEF",
						"--jump", "ACCEPT"},
					{"-s", "9.9.0.0/16",
						"-d", "5.6.7.8",
						"-m", "limit", "--limit", "2/min",
						"--jump", "LOG", "--log-prefix", "DENY_C2C_some-container-handle"},
					{"-s", "9.9.0.0/16",
						"-d", "5.6.7.8",
						"--jump", "REJECT",
						"--reject-with", "icmp-port-unreachable"},
				}))
			})
		})

		Context("when dns servers are specified", func() {
			It("creates rules for the dns servers", func() {
				err := netOut.Initialize("some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16", []string{"8.8.4.4", "1.2.3.4"})
				Expect(err).NotTo(HaveOccurred())
				Expect(ipTables.BulkAppendCallCount()).To(Equal(4))

				table, chain, rulespec := ipTables.BulkAppendArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("input-some-container-handle"))
				Expect(rulespec).To(Equal([]rules.IPTablesRule{
					{"-s", "5.6.7.8",
						"-m", "state", "--state", "RELATED,ESTABLISHED",
						"--jump", "ACCEPT"},
					{"-s", "5.6.7.8", "-p", "tcp", "-d", "8.8.4.4", "--destination-port", "53", "--jump", "ACCEPT"},
					{"-s", "5.6.7.8", "-p", "udp", "-d", "8.8.4.4", "--destination-port", "53", "--jump", "ACCEPT"},
					{"-s", "5.6.7.8", "-p", "tcp", "-d", "1.2.3.4", "--destination-port", "53", "--jump", "ACCEPT"},
					{"-s", "5.6.7.8", "-p", "udp", "-d", "1.2.3.4", "--destination-port", "53", "--jump", "ACCEPT"},
					{"-s", "5.6.7.8",
						"--jump", "REJECT",
						"--reject-with", "icmp-port-unreachable"},
				}))

			})
		})
	})

	Describe("Cleanup", func() {
		It("deletes the correct jump rules from the forward chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(3))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("overlay"))
			Expect(handle).To(Equal("some-container-handle"))

			prefix, handle = chainNamer.PrefixArgsForCall(1)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			prefix, handle = chainNamer.PrefixArgsForCall(2)
			Expect(prefix).To(Equal("input"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(chainNamer.PostfixCallCount()).To(Equal(1))
			body, suffix := chainNamer.PostfixArgsForCall(0)
			Expect(body).To(Equal("netout-some-container-handle"))
			Expect(suffix).To(Equal("log"))

			Expect(ipTables.DeleteCallCount()).To(Equal(3))
			table, chain, extraArgs := ipTables.DeleteArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(extraArgs).To(Equal(rules.IPTablesRule{"--jump", "overlay-some-container-handle"}))

			table, chain, extraArgs = ipTables.DeleteArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(extraArgs).To(Equal(rules.IPTablesRule{"--jump", "netout-some-container-handle"}))

			table, chain, extraArgs = ipTables.DeleteArgsForCall(2)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("INPUT"))
			Expect(extraArgs).To(Equal(rules.IPTablesRule{"--jump", "input-some-container-handle"}))
		})

		It("clears the container chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.ClearChainCallCount()).To(Equal(4))
			table, chain := ipTables.ClearChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("overlay-some-container-handle"))

			table, chain = ipTables.ClearChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout-some-container-handle"))

			table, chain = ipTables.ClearChainArgsForCall(2)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("input-some-container-handle"))

			table, chain = ipTables.ClearChainArgsForCall(3)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-other-chain-name"))

		})

		It("deletes the container chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.DeleteChainCallCount()).To(Equal(4))
			table, chain := ipTables.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("overlay-some-container-handle"))

			table, chain = ipTables.DeleteChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout-some-container-handle"))

			table, chain = ipTables.DeleteChainArgsForCall(2)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("input-some-container-handle"))

			table, chain = ipTables.DeleteChainArgsForCall(3)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-other-chain-name"))

		})

		Context("when the chain namer fails", func() {
			BeforeEach(func() {
				chainNamer.PostfixReturns("", errors.New("banana"))
			})
			It("returns the error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError("getting chain name: banana"))
			})
		})

		Context("when deleting the jump rule fails", func() {
			BeforeEach(func() {
				ipTables.DeleteReturns(errors.New("yukon potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError(ContainSubstring("delete rule: yukon potato")))
			})
		})

		Context("when clearing the container chain fails", func() {
			BeforeEach(func() {
				ipTables.ClearChainReturns(errors.New("idaho potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError(ContainSubstring("clear chain: idaho potato")))
			})
		})

		Context("when deleting the container chain fails", func() {
			BeforeEach(func() {
				ipTables.DeleteChainReturns(errors.New("purple potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError(ContainSubstring("delete chain: purple potato")))
			})
		})

		Context("when all the steps fail", func() {
			BeforeEach(func() {
				ipTables.DeleteReturns(errors.New("yukon potato"))
				ipTables.ClearChainReturns(errors.New("idaho potato"))
				ipTables.DeleteChainReturns(errors.New("purple potato"))
			})
			It("returns all the errors", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError(ContainSubstring("delete rule: yukon potato")))
				Expect(err).To(MatchError(ContainSubstring("clear chain: idaho potato")))
				Expect(err).To(MatchError(ContainSubstring("delete chain: purple potato")))
			})
		})
	})

	Describe("InsertRule", func() {
		var (
			netOutRule     garden.NetOutRule
			convertedRules []rules.IPTablesRule
		)

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
			convertedRules = []rules.IPTablesRule{
				rules.IPTablesRule{"rule1"},
				rules.IPTablesRule{"rule2"},
			}
			converter.ConvertReturns(convertedRules)
		})

		It("prepends allow rules to the container's netout chain", func() {
			err := netOut.InsertRule("some-container-handle", netOutRule, "1.2.3.4")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(1))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(chainNamer.PostfixCallCount()).To(Equal(1))
			body, suffix := chainNamer.PostfixArgsForCall(0)
			Expect(body).To(Equal("netout-some-container-handle"))
			Expect(suffix).To(Equal("log"))

			Expect(converter.ConvertCallCount()).To(Equal(1))
			rule, ip, logChainName, logging := converter.ConvertArgsForCall(0)
			Expect(rule).To(Equal(netOutRule))
			Expect(ip).To(Equal("1.2.3.4"))
			Expect(logChainName).To(Equal("some-other-chain-name"))
			Expect(logging).To(Equal(false))

			Expect(ipTables.BulkInsertCallCount()).To(Equal(1))
			table, chain, pos, rulespec := ipTables.BulkInsertArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout-some-container-handle"))
			Expect(pos).To(Equal(1))
			Expect(rulespec).To(Equal(convertedRules))
		})

		Context("when the chain namer fails", func() {
			BeforeEach(func() {
				chainNamer.PostfixReturns("", errors.New("banana"))
			})
			It("returns the error", func() {
				err := netOut.InsertRule("some-container-handle", netOutRule, "1.2.3.4")
				Expect(err).To(MatchError("getting chain name: banana"))
			})
		})

		Context("when insert rule fails", func() {
			BeforeEach(func() {
				ipTables.BulkInsertReturns(errors.New("potato"))
			})
			It("returns an error", func() {
				err := netOut.InsertRule("some-container-handle", netOutRule, "1.2.3.4")
				Expect(err).To(MatchError("inserting net-out rule: potato"))
			})
		})

		Context("when the global logging is enabled", func() {
			BeforeEach(func() {
				netOut.ASGLogging = true
			})
			It("calls Convert with globalLogging set to true", func() {
				err := netOut.InsertRule("some-container-handle", netOutRule, "1.2.3.4")
				Expect(err).NotTo(HaveOccurred())

				Expect(converter.ConvertCallCount()).To(Equal(1))
				rule, ip, logChainName, logging := converter.ConvertArgsForCall(0)
				Expect(rule).To(Equal(netOutRule))
				Expect(ip).To(Equal("1.2.3.4"))
				Expect(logChainName).To(Equal("some-other-chain-name"))
				Expect(logging).To(Equal(true))
			})
		})
	})

	Describe("BulkInsertRules", func() {
		var (
			netOutRules  []garden.NetOutRule
			genericRules []rules.IPTablesRule
		)

		BeforeEach(func() {
			genericRules = []rules.IPTablesRule{
				rules.IPTablesRule{"rule1"},
				rules.IPTablesRule{"rule2"},
			}

			converter.BulkConvertReturns(genericRules)

		})

		It("prepends allow rules to the container's netout chain", func() {
			err := netOut.BulkInsertRules("some-container-handle", netOutRules, "1.2.3.4")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(1))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(chainNamer.PostfixCallCount()).To(Equal(1))
			body, suffix := chainNamer.PostfixArgsForCall(0)
			Expect(body).To(Equal("netout-some-container-handle"))
			Expect(suffix).To(Equal("log"))

			Expect(converter.BulkConvertCallCount()).To(Equal(1))
			convertedRules, ip, logChainName, logging := converter.BulkConvertArgsForCall(0)
			Expect(convertedRules).To(Equal(netOutRules))
			Expect(ip).To(Equal("1.2.3.4"))
			Expect(logChainName).To(Equal("some-other-chain-name"))
			Expect(logging).To(Equal(false))

			Expect(ipTables.BulkInsertCallCount()).To(Equal(1))
			table, chain, pos, rulespec := ipTables.BulkInsertArgsForCall(0)

			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout-some-container-handle"))
			Expect(pos).To(Equal(1))
			Expect(rulespec).To(Equal(genericRules))
		})

		Context("when the chain namer fails", func() {
			BeforeEach(func() {
				chainNamer.PostfixReturns("", errors.New("banana"))
			})
			It("returns the error", func() {
				err := netOut.BulkInsertRules("some-container-handle", netOutRules, "1.2.3.4")
				Expect(err).To(MatchError("getting chain name: banana"))
			})
		})

		Context("when bulk insert fails", func() {
			BeforeEach(func() {
				ipTables.BulkInsertReturns(errors.New("potato"))
			})
			It("returns an error", func() {
				err := netOut.BulkInsertRules("some-container-handle", netOutRules, "1.2.3.4")
				Expect(err).To(MatchError("bulk inserting net-out rules: potato"))
			})
		})

		Context("when the global logging is enabled", func() {
			BeforeEach(func() {
				netOut.ASGLogging = true
			})
			It("calls BulkConvert with globalLogging set to true", func() {
				err := netOut.BulkInsertRules("some-container-handle", netOutRules, "1.2.3.4")
				Expect(err).NotTo(HaveOccurred())

				Expect(converter.BulkConvertCallCount()).To(Equal(1))
				convertedRules, ip, logChainName, logging := converter.BulkConvertArgsForCall(0)
				Expect(convertedRules).To(Equal(netOutRules))
				Expect(ip).To(Equal("1.2.3.4"))
				Expect(logChainName).To(Equal("some-other-chain-name"))
				Expect(logging).To(Equal(true))
			})
		})
	})
})
