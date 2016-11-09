package rules_test

import (
	"errors"
	"lib/fakes"
	"lib/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LockedIptables", func() {
	var (
		lockedIPT *rules.LockedIPTables
		ipt       *fakes.IPTables
		lock      *fakes.Locker
		rulespec  []string
	)
	BeforeEach(func() {
		ipt = &fakes.IPTables{}
		lock = &fakes.Locker{}
		lockedIPT = &rules.LockedIPTables{
			IPTables: ipt,
			Locker:   lock,
		}
		rulespec = []string{"some", "args"}
	})
	Describe("Exists", func() {
		BeforeEach(func() {
			ipt.ExistsReturns(true, nil)
		})
		It("passes the correct parameters to the iptables library", func() {
			exists, err := lockedIPT.Exists("some-table", "some-chain", rulespec...)
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
				_, err := lockedIPT.Exists("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.ExistsReturns(false, errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.Exists("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.ExistsReturns(false, errors.New("patato"))
			})
			It("returns an error", func() {
				_, err := lockedIPT.Exists("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("Insert", func() {
		It("locks and passes the correct parameters to the iptables library", func() {
			err := lockedIPT.Insert("some-table", "some-chain", 2, rulespec...)
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.InsertCallCount()).To(Equal(1))
			table, chain, pos, spec := ipt.InsertArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
			Expect(pos).To(Equal(2))
			Expect(spec).To(Equal(rulespec))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.Insert("some-table", "some-chain", 2, rulespec...)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.InsertReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.Insert("some-table", "some-chain", 2, rulespec...)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.InsertReturns(errors.New("patato"))
			})
			It("returns an error", func() {
				err := lockedIPT.Insert("some-table", "some-chain", 2, rulespec...)
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("AppendUnique", func() {
		It("locks and passes the correct parameters to the iptables library", func() {
			err := lockedIPT.AppendUnique("some-table", "some-chain", rulespec...)
			Expect(err).NotTo(HaveOccurred())

			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
			Expect(ipt.AppendUniqueCallCount()).To(Equal(1))
			table, chain, spec := ipt.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
			Expect(spec).To(Equal(rulespec))
		})

		Context("when locking fails", func() {
			BeforeEach(func() {
				lock.LockReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.AppendUnique("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.AppendUniqueReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.AppendUnique("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.AppendUniqueReturns(errors.New("patato"))
			})
			It("returns an error", func() {
				err := lockedIPT.AppendUnique("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("iptables call: patato and unlock: banana"))
			})
		})
	})

	Describe("Delete", func() {
		It("locks and passes the correct parameters to the iptables library", func() {
			err := lockedIPT.Delete("some-table", "some-chain", rulespec...)
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
				err := lockedIPT.Delete("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("lock: banana"))
			})
		})

		Context("when iptables call fails and unlock succeeds", func() {
			BeforeEach(func() {
				ipt.DeleteReturns(errors.New("banana"))
			})
			It("returns an error", func() {
				err := lockedIPT.Delete("some-table", "some-chain", rulespec...)
				Expect(err).To(MatchError("iptables call: banana and unlock: <nil>"))
			})
		})

		Context("when iptables call fails and unlock fails", func() {
			BeforeEach(func() {
				lock.UnlockReturns(errors.New("banana"))
				ipt.DeleteReturns(errors.New("patato"))
			})
			It("returns an error", func() {
				err := lockedIPT.Delete("some-table", "some-chain", rulespec...)
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
