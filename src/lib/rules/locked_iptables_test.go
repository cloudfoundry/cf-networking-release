package rules_test

import (
	"errors"
	"fmt"
	"lib/fakes"
	"lib/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LockedIptables", func() {
	var (
		lockedIPT *rules.LockedIPTables
		ipt       *fakes.IPTables
		restorer  *fakes.Restorer
		lock      *fakes.Locker
		rulespec  []string
		rule      rules.IPTablesRule
	)
	BeforeEach(func() {
		ipt = &fakes.IPTables{}
		lock = &fakes.Locker{}
		restorer = &fakes.Restorer{}
		lockedIPT = &rules.LockedIPTables{
			IPTables: ipt,
			Locker:   lock,
			Restorer: restorer,
		}
		rulespec = []string{"some", "args"}
		rule = rules.IPTablesRule{"some", "args"}
	})
	Describe("BulkInsert", func() {
		var ruleSet []rules.IPTablesRule
		BeforeEach(func() {
			ruleSet = []rules.IPTablesRule{
				rules.NewMarkSetRule("1.2.3.4", "A", "a-guid"),
				rules.NewMarkSetRule("2.2.2.2", "B", "b-guid"),
			}
		})

		It("constructs the input and passes it to the restorer", func() {
			err := lockedIPT.BulkInsert("some-table", "some-chain", 1, ruleSet...)
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(restorer.RestoreCallCount()).To(Equal(1))
			restoreInput := restorer.RestoreArgsForCall(0)
			Expect(restoreInput).To(ContainSubstring("*some-table\n"))
			Expect(restoreInput).To(ContainSubstring("-I some-chain 1 --source 1.2.3.4 --jump MARK --set-xmark 0xA -m comment --comment src:a-guid\n"))
			Expect(restoreInput).To(ContainSubstring("-I some-chain 1 --source 2.2.2.2 --jump MARK --set-xmark 0xB -m comment --comment src:b-guid\n"))
			Expect(restoreInput).To(ContainSubstring("COMMIT\n"))
		})
		Context("when the lock fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkInsert("some-table", "some-chain", 1, ruleSet...)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when the restorer fails", func() {
			BeforeEach(func() {
				restorer.RestoreReturns(fmt.Errorf("banana"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkInsert("some-table", "some-chain", 1, ruleSet...)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when the unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkInsert("some-table", "some-chain", 1, ruleSet...)
				Expect(err).To(MatchError("banana"))
			})
		})
		Context("when the restorer fails and then the unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				restorer.RestoreReturns(fmt.Errorf("patato"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkInsert("some-table", "some-chain", 1, ruleSet...)
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("BulkAppend", func() {
		var ruleSet []rules.IPTablesRule
		BeforeEach(func() {
			ruleSet = []rules.IPTablesRule{
				rules.NewMarkSetRule("1.2.3.4", "A", "a-guid"),
				rules.NewMarkSetRule("2.2.2.2", "B", "b-guid"),
			}
		})

		It("constructs the input and passes it to the restorer", func() {
			err := lockedIPT.BulkAppend("some-table", "some-chain", ruleSet...)
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(restorer.RestoreCallCount()).To(Equal(1))
			restoreInput := restorer.RestoreArgsForCall(0)
			Expect(restoreInput).To(ContainSubstring("*some-table\n"))
			Expect(restoreInput).To(ContainSubstring("-A some-chain --source 1.2.3.4 --jump MARK --set-xmark 0xA -m comment --comment src:a-guid\n"))
			Expect(restoreInput).To(ContainSubstring("-A some-chain --source 2.2.2.2 --jump MARK --set-xmark 0xB -m comment --comment src:b-guid\n"))
			Expect(restoreInput).To(ContainSubstring("COMMIT\n"))
		})
		Context("when the lock fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkAppend("some-table", "some-chain", ruleSet...)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when the restorer fails", func() {
			BeforeEach(func() {
				restorer.RestoreReturns(fmt.Errorf("banana"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkAppend("some-table", "some-chain", ruleSet...)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when the unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkAppend("some-table", "some-chain", ruleSet...)
				Expect(err).To(MatchError("banana"))
			})
		})
		Context("when the restorer fails and then the unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				restorer.RestoreReturns(fmt.Errorf("patato"))
			})
			It("should return an error", func() {
				err := lockedIPT.BulkAppend("some-table", "some-chain", ruleSet...)
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("Exists", func() {
		BeforeEach(func() {
			ipt.ExistsReturns(true, nil)
		})
		It("passes the correct parameters to the iptables library", func() {
			exists, err := lockedIPT.Exists("some-table", "some-chain", rule)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(Equal(true))

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.ExistsCallCount()).To(Equal(1))
			table, chain, spec := ipt.ExistsArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
			Expect(spec).To(Equal(rulespec))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.Exists("some-table", "some-chain", rule)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.ExistsReturns(false, errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.Exists("some-table", "some-chain", rule)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.ExistsReturns(false, errors.New("patato"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.Exists("some-table", "some-chain", rule)
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("Delete", func() {
		It("locks and passes the correct parameters to the iptables library", func() {
			err := lockedIPT.Delete("some-table", "some-chain", rule)
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.DeleteCallCount()).To(Equal(1))
			table, chain, spec := ipt.DeleteArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
			Expect(spec).To(Equal(rulespec))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.Delete("some-table", "some-chain", rule)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.DeleteReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.Delete("some-table", "some-chain", rule)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.DeleteReturns(errors.New("patato"))
			})
			It("returns an error", func() {
				err := lockedIPT.Delete("some-table", "some-chain", rule)
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			ipt.ListReturns([]string{"some", "list"}, nil)
		})
		It("locks and passes the correct parameters to the iptables library", func() {
			list, err := lockedIPT.List("some-table", "some-chain")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(Equal([]string{"some", "list"}))

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.ListCallCount()).To(Equal(1))
			table, chain := ipt.ListArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.List("some-table", "some-chain")
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.ListReturns(nil, errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.List("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.ListReturns(nil, errors.New("patato"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.List("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("NewChain", func() {
		It("locks and passes the correct parameters to the iptables library", func() {
			err := lockedIPT.NewChain("some-table", "some-chain")
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.NewChainCallCount()).To(Equal(1))
			table, chain := ipt.NewChainArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.NewChain("some-table", "some-chain")
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.NewChainReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.NewChain("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.NewChainReturns(errors.New("patato"))
			})
			It("returns an error", func() {
				err := lockedIPT.NewChain("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("DeleteChain", func() {
		It("locks and passes the correct parameters to the iptables library", func() {
			err := lockedIPT.DeleteChain("some-table", "some-chain")
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.DeleteChainCallCount()).To(Equal(1))
			table, chain := ipt.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.DeleteChain("some-table", "some-chain")
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.DeleteChainReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.DeleteChain("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.DeleteChainReturns(errors.New("patato"))
			})
			It("returns an error", func() {
				err := lockedIPT.DeleteChain("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("ClearChain", func() {
		It("locks and passes the correct parameters to the iptables library", func() {
			err := lockedIPT.ClearChain("some-table", "some-chain")
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.ClearChainCallCount()).To(Equal(1))
			table, chain := ipt.ClearChainArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.ClearChain("some-table", "some-chain")
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.ClearChainReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.ClearChain("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.ClearChainReturns(errors.New("patato"))
			})
			It("returns an error", func() {
				err := lockedIPT.ClearChain("some-table", "some-chain")
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})
})
