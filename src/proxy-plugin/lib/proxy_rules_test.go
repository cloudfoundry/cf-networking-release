package lib_test

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"lib/fakes"
	"lib/rules"
	"proxy-plugin/lib"
)

var _ = Describe("ProxyRules", func() {
	var (
		proxyRules lib.ProxyRules
		ipTables   *fakes.IPTablesAdapter
	)

	BeforeEach(func() {
		ipTables = &fakes.IPTablesAdapter{}
		proxyRules = lib.ProxyRules{
			IPTables:       ipTables,
			ProxyPort:      8090,
			OverlayNetwork: "10.255.0.0/16",
		}
	})

	Describe("Add", func() {
		It("adds proxy rules to the specified namespace", func() {
			err := proxyRules.Add("chain-name")
			Expect(err).ToNot(HaveOccurred())

			Expect(ipTables.NewChainCallCount()).To(Equal(1))
			table, chain := ipTables.NewChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("chain-name"))

			Expect(ipTables.BulkAppendCallCount()).To(Equal(1))
			table, chain, chainRules := ipTables.BulkAppendArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("chain-name"))
			Expect(chainRules).To(Equal([]rules.IPTablesRule{
				{"OUTPUT", "-j", "chain-name"},
				{"chain-name", "-m", "owner", "!", "--uid-owner", "1000", "-j", "RETURN"},
				{"chain-name", "-d", "10.255.0.0/16", "-p", "tcp", "-j", "REDIRECT", "--to-ports", "8090"},
			}))
		})

		Context("when creating a chain fails", func() {
			It("returns an error", func() {
				ipTables.NewChainReturns(errors.New("meow"))
				err := proxyRules.Add("chain-name")
				Expect(err).To(Equal(errors.New("creating new chain: meow")))
			})
		})

		Context("when bulk appending rules fails", func() {
			It("returns an error", func() {
				ipTables.BulkAppendReturns(errors.New("meow"))
				err := proxyRules.Add("chain-name")
				Expect(err).To(Equal(errors.New("appending rules: meow")))
			})
		})
	})

	Describe("Del", func() {
		It("removes the proxy chain from the specified namespace", func() {
			err := proxyRules.Del("chain-name")
			Expect(err).ToNot(HaveOccurred())

			Expect(ipTables.DeleteCallCount()).To(Equal(3))

			table, chain, rule := ipTables.DeleteArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("chain-name"))
			Expect(rule).To(Equal(rules.IPTablesRule{"OUTPUT", "-j", "chain-name"}))

			table, chain, rule = ipTables.DeleteArgsForCall(1)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("chain-name"))
			Expect(rule).To(Equal(rules.IPTablesRule{"chain-name", "-m", "owner", "!", "--uid-owner", "1000", "-j", "RETURN"}))

			table, chain, rule = ipTables.DeleteArgsForCall(2)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("chain-name"))
			Expect(rule).To(Equal(rules.IPTablesRule{"chain-name", "-d", "10.255.0.0/16", "-p", "tcp", "-j", "REDIRECT", "--to-ports", "8090"}))

			Expect(ipTables.DeleteChainCallCount()).To(Equal(1))
			table, chain = ipTables.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("chain-name"))
		})

		It("deletes the rules before it deletes the chain", func() {
			invocations := []string{}
			ipTables.DeleteStub = func(table, chain string, rulespec rules.IPTablesRule) error {
				invocations = append(invocations, "Delete")
				return nil
			}

			ipTables.DeleteChainStub = func(table, chain string) error {
				invocations = append(invocations, "Delete Chain")
				return nil
			}

			err := proxyRules.Del("chain-name")
			Expect(err).ToNot(HaveOccurred())
			Expect(invocations).To(Equal([]string{"Delete", "Delete", "Delete", "Delete Chain"}))
		})

		Context("when deleting the rule fails", func() {
			BeforeEach(func() {
				ipTables.DeleteReturns(errors.New("sneaky-cuddlebug"))
			})
			It("returns an error", func() {
				err := proxyRules.Del("chain-name")
				Expect(err).To(Equal(errors.New("deleting rule: sneaky-cuddlebug")))
			})
		})

		Context("when deleting the chain fails", func() {
			BeforeEach(func() {
				ipTables.DeleteChainReturns(errors.New("sneaky-cuddlebug"))
			})

			It("returns an error", func() {
				err := proxyRules.Del("chain-name")
				Expect(err).To(Equal(errors.New("deleting chain: sneaky-cuddlebug")))
			})
		})
	})
})
