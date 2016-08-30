package legacynet_test

import (
	"errors"
	"garden-external-networker/fakes"
	"garden-external-networker/legacynet"

	lib_fakes "lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Netin", func() {

	var (
		netIn      *legacynet.NetIn
		ipTables   *lib_fakes.IPTables
		chainNamer *fakes.ChainNamer
	)

	BeforeEach(func() {
		ipTables = &lib_fakes.IPTables{}
		chainNamer = &fakes.ChainNamer{}
		netIn = &legacynet.NetIn{
			ChainNamer: chainNamer,
			IPTables:   ipTables,
		}
		chainNamer.NameReturns("some-chain-name")
	})

	Describe("Initialize", func() {
		It("creates the chain with the name from the chain namer", func() {
			err := netIn.Initialize("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.NameCallCount()).To(Equal(1))
			prefix, handle := chainNamer.NameArgsForCall(0)
			Expect(prefix).To(Equal("netin"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.NewChainCallCount()).To(Equal(1))
			_, chain := ipTables.NewChainArgsForCall(0)
			Expect(chain).To(Equal("some-chain-name"))
		})

		It("adds a jump rule for the new chain", func() {
			err := netIn.Initialize("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.AppendUniqueCallCount()).To(Equal(1))
			table, chain, extraArgs := ipTables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("PREROUTING"))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-chain-name"}))
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

		Context("when appending the jump rule fails", func() {
			BeforeEach(func() {
				ipTables.AppendUniqueReturns(errors.New("sweet potato"))
			})
			It("returns an error", func() {
				err := netIn.Initialize("some-container-handle")
				Expect(err).To(MatchError("inserting rule: sweet potato"))
			})
		})
	})

	Describe("Cleanup", func() {
		It("deletes the correct jump rule from the prerouting chain", func() {
			err := netIn.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.NameCallCount()).To(Equal(1))
			prefix, handle := chainNamer.NameArgsForCall(0)
			Expect(prefix).To(Equal("netin"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.DeleteCallCount()).To(Equal(1))
			table, chain, extraArgs := ipTables.DeleteArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("PREROUTING"))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-chain-name"}))
		})

		It("clears the container chain", func() {
			err := netIn.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.ClearChainCallCount()).To(Equal(1))
			table, chain := ipTables.ClearChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("some-chain-name"))
		})

		It("deletes the container chain", func() {
			err := netIn.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.DeleteChainCallCount()).To(Equal(1))
			table, chain := ipTables.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("some-chain-name"))
		})

		Context("when deleting the jump rule fails", func() {
			BeforeEach(func() {
				ipTables.DeleteReturns(errors.New("yukon potato"))
			})
			It("returns an error", func() {
				err := netIn.Cleanup("some-container-handle")
				Expect(err).To(MatchError("delete rule: yukon potato"))
			})
		})

		Context("when clearing the container chain fails", func() {
			BeforeEach(func() {
				ipTables.ClearChainReturns(errors.New("idaho potato"))
			})
			It("returns an error", func() {
				err := netIn.Cleanup("some-container-handle")
				Expect(err).To(MatchError("clear chain: idaho potato"))
			})
		})

		Context("when deleting the container chain fails", func() {
			BeforeEach(func() {
				ipTables.DeleteChainReturns(errors.New("purple potato"))
			})
			It("returns an error", func() {
				err := netIn.Cleanup("some-container-handle")
				Expect(err).To(MatchError("delete chain: purple potato"))
			})
		})
	})

	Describe("AddRule", func() {
		It("creates and enforces a netin rule", func() {
			err := netIn.AddRule("some-container-handle", 1111, 2222, "1.2.3.4", "5.6.7.8")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.NameCallCount()).To(Equal(1))
			prefix, handle := chainNamer.NameArgsForCall(0)
			Expect(prefix).To(Equal("netin"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.AppendUniqueCallCount()).To(Equal(1))
			table, chain, extraArgs := ipTables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("some-chain-name"))
			Expect(extraArgs).To(Equal([]string{
				"-d", "1.2.3.4", "-p", "tcp",
				"-m", "tcp", "--dport", "1111",
				"--jump", "DNAT",
				"--to-destination", "5.6.7.8:2222",
			}))
		})

		Context("when writing the netin rule fails", func() {
			BeforeEach(func() {
				ipTables.AppendUniqueReturns(errors.New("blue potato"))
			})
			It("returns an error", func() {
				err := netIn.AddRule("some-container-handle", 1111, 2222, "1.2.3.4", "5.6.7.8")
				Expect(err).To(MatchError("inserting rule: blue potato"))
			})
		})
	})
})
