package legacynet_test

import (
	"errors"
	"garden-external-networker/fakes"
	"garden-external-networker/legacynet"
	"net"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager/lagertest"

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
		ipTables   *lib_fakes.IPTablesExtended
		logger     *lagertest.TestLogger
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		chainNamer = &fakes.ChainNamer{}
		converter = &fakes.NetOutRuleConverter{}
		ipTables = &lib_fakes.IPTablesExtended{}
		netOut = &legacynet.NetOut{
			ChainNamer: chainNamer,
			IPTables:   ipTables,
			Converter:  converter,
		}
		chainNamer.PrefixReturns("some-chain-name")
		chainNamer.PostfixReturns("some-other-chain-name", nil)
	})

	Describe("Initialize", func() {
		It("creates the netout chain and the logging chain", func() {
			err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(1))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(chainNamer.PostfixCallCount()).To(Equal(1))
			body, suffix := chainNamer.PostfixArgsForCall(0)
			Expect(body).To(Equal("some-chain-name"))
			Expect(suffix).To(Equal("log"))

			Expect(ipTables.NewChainCallCount()).To(Equal(2))
			table, chain := ipTables.NewChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))
			table, chain = ipTables.NewChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-other-chain-name"))
		})

		It("inserts a jump rule for the new chains", func() {
			err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.InsertCallCount()).To(Equal(2))
			table, chain, position, extraArgs := ipTables.InsertArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(position).To(Equal(1))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-chain-name"}))

			table, chain, position, extraArgs = ipTables.InsertArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(position).To(Equal(1))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-other-chain-name"}))
		})

		It("writes the default netout and logging rules", func() {
			err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.BulkAppendCallCount()).To(Equal(2))

			table, chain, rulespec := ipTables.BulkAppendArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{
				{"-s", "5.6.7.8",
					"!", "-d", "9.9.0.0/16",
					"-m", "state", "--state", "RELATED,ESTABLISHED",
					"--jump", "RETURN"},
				{"-s", "5.6.7.8",
					"!", "-d", "9.9.0.0/16",
					"--jump", "REJECT",
					"--reject-with", "icmp-port-unreachable"},
			}))

			table, chain, rulespec = ipTables.BulkAppendArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-other-chain-name"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{
				{"-p", "tcp",
					"-m", "conntrack", "--ctstate", "INVALID,NEW,UNTRACKED",
					"-j", "LOG", "--log-prefix", "some-container-handle"},
				{"--jump", "RETURN"},
			}))
		})

		It("writes the logging rules", func() {
			err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when creating a new chain fails", func() {
			BeforeEach(func() {
				ipTables.NewChainReturns(errors.New("potata"))
			})
			It("returns the error", func() {
				err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
				Expect(err).To(MatchError("creating chain: potata"))
			})
		})

		Context("when the chain namer fails", func() {
			BeforeEach(func() {
				chainNamer.PostfixReturns("", errors.New("banana"))
			})
			It("returns the error", func() {
				err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
				Expect(err).To(MatchError("getting chain name: banana"))
			})
		})

		Context("when inserting a new rule fails", func() {
			BeforeEach(func() {
				ipTables.InsertReturns(errors.New("potato"))
			})
			It("returns the error", func() {
				err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
				Expect(err).To(MatchError("inserting rule: potato"))
			})
		})

		Context("when writing the netout rule fails", func() {
			BeforeEach(func() {
				ipTables.BulkAppendReturns(errors.New("potato"))
			})
			It("returns the error", func() {
				err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
				Expect(err).To(MatchError("appending rule: potato"))
			})
		})
	})

	Describe("Cleanup", func() {
		It("deletes the correct jump rules from the forward chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(1))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(chainNamer.PostfixCallCount()).To(Equal(1))
			body, suffix := chainNamer.PostfixArgsForCall(0)
			Expect(body).To(Equal("some-chain-name"))
			Expect(suffix).To(Equal("log"))

			Expect(ipTables.DeleteCallCount()).To(Equal(2))
			table, chain, extraArgs := ipTables.DeleteArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-chain-name"}))

			table, chain, extraArgs = ipTables.DeleteArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-other-chain-name"}))

		})

		It("clears the container chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.ClearChainCallCount()).To(Equal(2))
			table, chain := ipTables.ClearChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))

			table, chain = ipTables.ClearChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-other-chain-name"))

		})

		It("deletes the container chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.DeleteChainCallCount()).To(Equal(2))
			table, chain := ipTables.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))

			table, chain = ipTables.DeleteChainArgsForCall(1)
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
				Expect(err).To(MatchError("delete rule: yukon potato"))
			})
		})

		Context("when clearing the container chain fails", func() {
			BeforeEach(func() {
				ipTables.ClearChainReturns(errors.New("idaho potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError("clear chain: idaho potato"))
			})
		})

		Context("when deleting the container chain fails", func() {
			BeforeEach(func() {
				ipTables.DeleteChainReturns(errors.New("purple potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError("delete chain: purple potato"))
			})
		})
	})

	Describe("InsertRule", func() {
		var netOutRule garden.NetOutRule

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
			converter.ConvertReturns([]rules.IPTablesRule{
				rules.IPTablesRule{"rule1"},
				rules.IPTablesRule{"rule2"},
			})
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
			Expect(body).To(Equal("some-chain-name"))
			Expect(suffix).To(Equal("log"))

			Expect(converter.ConvertCallCount()).To(Equal(1))
			rule, ip, logChainName := converter.ConvertArgsForCall(0)
			Expect(rule).To(Equal(netOutRule))
			Expect(ip).To(Equal("1.2.3.4"))
			Expect(logChainName).To(Equal("some-other-chain-name"))

			Expect(ipTables.InsertCallCount()).To(Equal(2))
			writtenRules := [][]string{}
			for i := 0; i < 2; i++ {
				table, chain, pos, rulespec := ipTables.InsertArgsForCall(i)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("some-chain-name"))
				Expect(pos).To(Equal(1))
				writtenRules = append(writtenRules, rulespec)
			}
			Expect(writtenRules).To(ConsistOf(
				[]string{"rule1"},
				[]string{"rule2"},
			))
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
				ipTables.InsertReturns(errors.New("potato"))
			})
			It("returns an error", func() {
				err := netOut.InsertRule("some-container-handle", netOutRule, "1.2.3.4")
				Expect(err).To(MatchError("inserting net-out rule: potato"))
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
			Expect(body).To(Equal("some-chain-name"))
			Expect(suffix).To(Equal("log"))

			Expect(converter.BulkConvertCallCount()).To(Equal(1))
			convertedRules, ip, logChainName := converter.BulkConvertArgsForCall(0)
			Expect(convertedRules).To(Equal(netOutRules))
			Expect(ip).To(Equal("1.2.3.4"))
			Expect(logChainName).To(Equal("some-other-chain-name"))

			Expect(ipTables.BulkInsertCallCount()).To(Equal(1))
			table, chain, pos, rulespec := ipTables.BulkInsertArgsForCall(0)

			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))
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
	})
})
