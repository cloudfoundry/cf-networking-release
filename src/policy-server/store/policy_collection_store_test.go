package store_test

import (
	"errors"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"
	"policy-server/store/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyCollectionStore", func() {
	var (
		mockDB                *fakes.Db
		policyStore           *fakes.Store
		tx                    *dbfakes.Transaction
		policyCollectionStore store.PolicyCollectionStore
		egressPolicyStore     *fakes.EgressPolicyStore
		policyCollection      store.PolicyCollection
	)

	BeforeEach(func() {
		mockDB = &fakes.Db{}
		policyStore = &fakes.Store{}
		egressPolicyStore = &fakes.EgressPolicyStore{}
		tx = &dbfakes.Transaction{}

		policyCollectionStore = store.PolicyCollectionStore{
			Conn:              mockDB,
			PolicyStore:       policyStore,
			EgressPolicyStore: egressPolicyStore,
		}

		mockDB.BeginxReturns(tx, nil)

		policyCollection = store.PolicyCollection{
			Policies: []store.Policy{
				{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Ports: store.Ports{
							Start: 8080,
							End:   9090,
						},
					},
				}, {
					Source: store.Source{ID: "another-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "udp",
						Ports: store.Ports{
							Start: 1234,
							End:   1234,
						},
					},
				},
			},
			EgressPolicies: []store.EgressPolicy{
				{
					Source: store.EgressSource{ID: "egress-source-id"},
					Destination: store.EgressDestination{
						Protocol: "tcp",
						IPRanges: []store.IPRange{{
							Start: "1.2.3.4",
							End:   "1.2.3.5",
						}},
					},
				},
				{
					Source: store.EgressSource{ID: "egress-source-id-2"},
					Destination: store.EgressDestination{
						Protocol: "udp",
						IPRanges: []store.IPRange{{
							Start: "1.2.3.7",
							End:   "1.2.3.8",
						}},
					},
				},
			},
		}
	})

	Describe("Create", func() {
		It("starts a transaction, defers to the policy store and the egress policy store, then commits", func() {
			Expect(policyCollectionStore.Create(policyCollection)).ToNot(HaveOccurred())
			Expect(policyStore.CreateWithTxCallCount()).To(Equal(1))
			passedTx, passedPolicies := policyStore.CreateWithTxArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedPolicies).To(Equal(policyCollection.Policies))

			Expect(egressPolicyStore.CreateWithTxCallCount()).To(Equal(1))
			passedTx, passedEgressPolicies := egressPolicyStore.CreateWithTxArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicies).To(Equal(policyCollection.EgressPolicies))

			Expect(tx.CommitCallCount()).To(Equal(1))
		})

		Context("when the transaction fails to begin", func() {
			It("returns an error", func() {
				mockDB.BeginxReturns(nil, errors.New("potato"))
				Expect(policyCollectionStore.Create(policyCollection)).To(MatchError("begin transaction: potato"))
			})

			It("does not commit the transaction", func() {
				mockDB.BeginxReturns(nil, errors.New("potato"))
				Expect(tx.CommitCallCount()).To(Equal(0))
			})
		})

		Context("when the policy store fails to create", func() {
			It("returns an error", func() {
				policyStore.CreateWithTxReturns(errors.New("failed to create policy"))
				Expect(policyCollectionStore.Create(policyCollection)).To(MatchError("failed to create policy"))
			})

			It("does not commit the transaction", func() {
				policyStore.CreateWithTxReturns(errors.New("failed to create policy"))
				Expect(tx.CommitCallCount()).To(Equal(0))
			})

			It("rolls back the changes", func() {
				policyStore.CreateWithTxReturns(errors.New("failed to create policy"))
				policyCollectionStore.Create(policyCollection)
				Expect(tx.RollbackCallCount()).To(Equal(1))
			})

			Context("when the rollback fails", func() {
				It("returns the original error wrapped with the rollback error", func() {
					policyStore.CreateWithTxReturns(errors.New("failed to create policy"))
					tx.RollbackReturns(errors.New("rollback failed it's all over folks"))
					Expect(policyCollectionStore.Create(policyCollection)).To(MatchError(
						"database rollback: rollback failed it's all over folks (sql error: failed to create policy)"))
				})
			})
		})

		Context("when the egress policy store fails to create", func() {
			It("returns an error", func() {
				egressPolicyStore.CreateWithTxReturns(errors.New("failed to create egress policy"))
				Expect(policyCollectionStore.Create(policyCollection)).To(MatchError("failed to create egress policy"))
			})

			It("does not commit the transaction", func() {
				egressPolicyStore.CreateWithTxReturns(errors.New("failed to create policy"))
				Expect(tx.CommitCallCount()).To(Equal(0))
			})

			It("rolls back the changes", func() {
				egressPolicyStore.CreateWithTxReturns(errors.New("failed to create policy"))
				policyCollectionStore.Create(policyCollection)
				Expect(tx.RollbackCallCount()).To(Equal(1))
			})

			Context("when the rollback fails", func() {
				It("returns the original error wrapped with the rollback error", func() {
					egressPolicyStore.CreateWithTxReturns(errors.New("failed to create policy"))
					tx.RollbackReturns(errors.New("rollback failed it's all over folks"))
					Expect(policyCollectionStore.Create(policyCollection)).To(MatchError(
						"database rollback: rollback failed it's all over folks (sql error: failed to create policy)"))
				})
			})
		})

		Context("when the commit fails", func() {
			It("returns an error", func() {
				tx.CommitReturns(errors.New("banana"))
				Expect(policyCollectionStore.Create(policyCollection)).To(MatchError("commit transaction: banana"))
			})
		})
	})

	Describe("Delete", func() {
		It("starts a transaction, defers to the policy store and egress policy store, then commits", func() {
			Expect(policyCollectionStore.Delete(policyCollection)).To(Succeed())
			Expect(policyStore.DeleteWithTxCallCount()).To(Equal(1))
			passedTx, passedPolicies := policyStore.DeleteWithTxArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedPolicies).To(Equal(policyCollection.Policies))
			Expect(tx.CommitCallCount()).To(Equal(1))

			passedTx, passedEgressPolicies := egressPolicyStore.DeleteWithTxArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicies).To(Equal(policyCollection.EgressPolicies))

		})

		Context("when the transaction fails to begin", func() {
			It("returns an error", func() {
				mockDB.BeginxReturns(nil, errors.New("potato"))
				Expect(policyCollectionStore.Delete(policyCollection)).To(MatchError("begin transaction: potato"))
			})

			It("does not commit the transaction", func() {
				mockDB.BeginxReturns(nil, errors.New("potato"))
				policyCollectionStore.Delete(policyCollection)
				Expect(tx.CommitCallCount()).To(Equal(0))
			})
		})

		Context("when the policy store fails to delete", func() {
			It("returns an error", func() {
				policyStore.DeleteWithTxReturns(errors.New("failed to delete policy"))
				Expect(policyCollectionStore.Delete(policyCollection)).To(MatchError("failed to delete policy"))
			})

			It("rolls back the transaction", func() {
				policyStore.DeleteWithTxReturns(errors.New("failed to delete policy"))
				policyCollectionStore.Delete(policyCollection)
				Expect(tx.RollbackCallCount()).To(Equal(1))
			})

			It("does not commit the transaction", func() {
				policyStore.DeleteWithTxReturns(errors.New("failed to delete policy"))
				policyCollectionStore.Delete(policyCollection)
				Expect(tx.CommitCallCount()).To(Equal(0))
			})
		})

		Context("when the egress policy store fails to delete", func() {
			It("returns an error", func() {
				egressPolicyStore.DeleteWithTxReturns(errors.New("failed to delete egress policy"))
				Expect(policyCollectionStore.Delete(policyCollection)).To(MatchError("failed to delete egress policy"))
			})

			It("rolls back the transaction", func() {
				egressPolicyStore.DeleteWithTxReturns(errors.New("failed to delete egress policy"))
				policyCollectionStore.Delete(policyCollection)
				Expect(tx.RollbackCallCount()).To(Equal(1))
			})

			It("does not commit the transaction", func() {
				egressPolicyStore.DeleteWithTxReturns(errors.New("failed to delete egress policy"))
				policyCollectionStore.Delete(policyCollection)
				Expect(tx.CommitCallCount()).To(Equal(0))
			})
		})

		Context("when the commit fails", func() {
			It("returns an error", func() {
				tx.CommitReturns(errors.New("banana"))
				Expect(policyCollectionStore.Delete(policyCollection)).To(MatchError("commit transaction: banana"))
			})
		})
	})

	Describe("All", func() {
		BeforeEach(func() {
			policyStore.AllReturns(policyCollection.Policies, nil)
			egressPolicyStore.AllReturns(policyCollection.EgressPolicies, nil)
		})

		It("aggregates calls to the policy store and egress policy store", func() {
			allPolicies, err := policyCollectionStore.All()
			Expect(err).ToNot(HaveOccurred())
			Expect(allPolicies).To(Equal(policyCollection))
		})

		It("returns the error from policy store", func() {
			policyStore.AllReturns(nil, errors.New("foxtrot"))

			_, err := policyCollectionStore.All()
			Expect(err).To(MatchError("foxtrot"))
		})

		It("returns the error from egress policy store", func() {
			egressPolicyStore.AllReturns(nil, errors.New("whiskey"))

			_, err := policyCollectionStore.All()
			Expect(err).To(MatchError("whiskey"))
		})
	})
})
