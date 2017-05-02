package legacynet_test

import (
	"cni-wrapper-plugin/fakes"
	"cni-wrapper-plugin/legacynet"
	"errors"

	lib_fakes "lib/fakes"
	"lib/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Netin", func() {

	var (
		netIn      *legacynet.NetIn
		ipTables   *lib_fakes.IPTablesAdapter
		chainNamer *fakes.ChainNamer
	)

	BeforeEach(func() {
		ipTables = &lib_fakes.IPTablesAdapter{}
		chainNamer = &fakes.ChainNamer{}
		netIn = &legacynet.NetIn{
			ChainNamer: chainNamer,
			IPTables:   ipTables,
			IngressTag: "FEEDBEEF",
		}
		chainNamer.PrefixReturns("some-chain-name")
	})

	Describe("Initialize", func() {
		It("creates the chain with the name from the chain namer in the nat and mangle tables", func() {
			err := netIn.Initialize("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(1))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("netin"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.NewChainCallCount()).To(Equal(2))
			table, chain := ipTables.NewChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("some-chain-name"))

			table, chain = ipTables.NewChainArgsForCall(1)
			Expect(table).To(Equal("mangle"))
			Expect(chain).To(Equal("some-chain-name"))
		})

		It("adds a jump rule for the new chain", func() {
			err := netIn.Initialize("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.BulkInsertCallCount()).To(Equal(2))
			table, chain, position, rulespec := ipTables.BulkInsertArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("PREROUTING"))
			Expect(position).To(Equal(1))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{{"--jump", "some-chain-name"}}))

			table, chain, position, rulespec = ipTables.BulkInsertArgsForCall(1)
			Expect(table).To(Equal("mangle"))
			Expect(chain).To(Equal("PREROUTING"))
			Expect(position).To(Equal(1))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{{"--jump", "some-chain-name"}}))
		})

		Context("when creating a new chain fails", func() {
			BeforeEach(func() {
				ipTables.NewChainReturns(errors.New("potato"))
			})
			It("returns an error", func() {
				err := netIn.Initialize("some-container-handle")
				Expect(err).To(MatchError("creating chain: potato"))
			})
		})

		Context("when adding the jump rule fails", func() {
			BeforeEach(func() {
				ipTables.BulkInsertReturns(errors.New("sweet potato"))
			})
			It("returns an error", func() {
				err := netIn.Initialize("some-container-handle")
				Expect(err).To(MatchError("inserting rule: sweet potato"))
			})
		})
	})

	Describe("Cleanup", func() {
		It("deletes the correct jump rule from the prerouting chain in both tables", func() {
			err := netIn.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(1))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("netin"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.DeleteCallCount()).To(Equal(2))
			table, chain, extraArgs := ipTables.DeleteArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("PREROUTING"))
			Expect(extraArgs).To(Equal(rules.IPTablesRule{"--jump", "some-chain-name"}))

			table, chain, extraArgs = ipTables.DeleteArgsForCall(1)
			Expect(table).To(Equal("mangle"))
			Expect(chain).To(Equal("PREROUTING"))
			Expect(extraArgs).To(Equal(rules.IPTablesRule{"--jump", "some-chain-name"}))
		})

		It("clears the container chain", func() {
			err := netIn.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.ClearChainCallCount()).To(Equal(2))
			table, chain := ipTables.ClearChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("some-chain-name"))

			table, chain = ipTables.ClearChainArgsForCall(1)
			Expect(table).To(Equal("mangle"))
			Expect(chain).To(Equal("some-chain-name"))
		})

		It("deletes the container chain", func() {
			err := netIn.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.DeleteChainCallCount()).To(Equal(2))
			table, chain := ipTables.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("some-chain-name"))

			table, chain = ipTables.DeleteChainArgsForCall(1)
			Expect(table).To(Equal("mangle"))
			Expect(chain).To(Equal("some-chain-name"))
		})

		Context("when deleting the jump rule fails", func() {
			BeforeEach(func() {
				ipTables.DeleteReturns(errors.New("yukon potato"))
			})
			It("returns an error", func() {
				err := netIn.Cleanup("some-container-handle")
				Expect(err).To(MatchError(ContainSubstring("delete rule: yukon potato")))
			})

			It("still attempts to clear the chain", func() {
				netIn.Cleanup("some-container-handle")
				Expect(ipTables.ClearChainCallCount()).To(Equal(2))
			})
		})

		Context("when clearing the container chain fails", func() {
			BeforeEach(func() {
				ipTables.ClearChainReturns(errors.New("idaho potato"))
			})
			It("returns an error", func() {
				err := netIn.Cleanup("some-container-handle")
				Expect(err).To(MatchError(ContainSubstring("clear chain: idaho potato")))
			})

			It("still attempts to delete the chain", func() {
				netIn.Cleanup("some-container-handle")
				Expect(ipTables.DeleteChainCallCount()).To(Equal(2))
			})
		})

		Context("when deleting the container chain fails", func() {
			BeforeEach(func() {
				ipTables.DeleteChainReturns(errors.New("purple potato"))
			})
			It("returns an error", func() {
				err := netIn.Cleanup("some-container-handle")
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
				err := netIn.Cleanup("some-container-handle")
				Expect(err).To(MatchError(ContainSubstring("delete rule: yukon potato")))
				Expect(err).To(MatchError(ContainSubstring("clear chain: idaho potato")))
				Expect(err).To(MatchError(ContainSubstring("delete chain: purple potato")))
			})
		})
	})

	Describe("AddRule", func() {
		It("creates and enforces a portforwarding and mark rule", func() {
			err := netIn.AddRule("some-container-handle", 1111, 2222, "1.2.3.4", "5.6.7.8")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.PrefixCallCount()).To(Equal(1))
			prefix, handle := chainNamer.PrefixArgsForCall(0)
			Expect(prefix).To(Equal("netin"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.BulkAppendCallCount()).To(Equal(2))
			table, chain, rulespec := ipTables.BulkAppendArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("some-chain-name"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{{
				"-d", "1.2.3.4", "-p", "tcp",
				"-m", "tcp", "--dport", "1111",
				"--jump", "DNAT",
				"--to-destination", "5.6.7.8:2222",
			}}))

			table, chain, rulespec = ipTables.BulkAppendArgsForCall(1)
			Expect(table).To(Equal("mangle"))
			Expect(chain).To(Equal("some-chain-name"))
			Expect(rulespec).To(Equal([]rules.IPTablesRule{{
				"-d", "1.2.3.4", "-p", "tcp",
				"-m", "tcp", "--dport", "1111",
				"--jump", "MARK",
				"--set-mark", "0xFEEDBEEF",
			}}))
		})

		Context("when writing the netin rule fails", func() {
			BeforeEach(func() {
				ipTables.BulkAppendReturns(errors.New("blue potato"))
			})
			It("returns an error", func() {
				err := netIn.AddRule("some-container-handle", 1111, 2222, "1.2.3.4", "5.6.7.8")
				Expect(err).To(MatchError("appending rule: blue potato"))
			})
		})

		Context("when the host ip is invalid", func() {
			It("returns an error", func() {
				err := netIn.AddRule("some-container-handle", 1111, 2222, "banana", "5.6.7.8")
				Expect(err).To(MatchError("invalid ip: banana"))
			})
		})

		Context("when the container ip is invalid", func() {
			It("returns an error", func() {
				err := netIn.AddRule("some-container-handle", 1111, 2222, "5.6.7.8", "banana")
				Expect(err).To(MatchError("invalid ip: banana"))
			})
		})
	})
})
