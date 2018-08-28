package store_test

import (
	"errors"
	"policy-server/db"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"
	"policy-server/store/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPolicyMetricsWrapper", func() {
	var (
		metricsWrapper    *store.EgressPolicyMetricsWrapper
		policies          []store.EgressPolicy
		srcGuids          []string
		fakeMetricsSender *fakes.MetricsSender
		fakeStore         *fakes.EgressPolicyStore
		tx                db.Transaction
	)

	BeforeEach(func() {
		tx = &dbfakes.Transaction{}
		fakeStore = &fakes.EgressPolicyStore{}
		fakeMetricsSender = &fakes.MetricsSender{}
		metricsWrapper = &store.EgressPolicyMetricsWrapper{
			Store:         fakeStore,
			MetricsSender: fakeMetricsSender,
		}
		policies = []store.EgressPolicy{{
			Source: store.EgressSource{ID: "some-app-guid"},
			Destination: store.EgressDestination{
				Protocol: "tcp",
				IPRanges: []store.IPRange{{Start: "8.0.8.0", End: "8.0.8.0"}},
			},
		}}
		srcGuids = []string{"some-app-guid"}
	})

	Describe("CreateWithTx", func() {
		It("calls CreateWithTx on the Store", func() {
			err := metricsWrapper.CreateWithTx(tx, policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.CreateWithTxCallCount()).To(Equal(1))
			passedTx, passedPolicies := fakeStore.CreateWithTxArgsForCall(0)
			Expect(passedPolicies).To(Equal(policies))
			Expect(passedTx).To(Equal(tx))
		})

		It("emits a metric", func() {
			err := metricsWrapper.CreateWithTx(tx, policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreCreateWithTxSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.CreateWithTxReturns(errors.New("banana"))
			})
			It("emits an error metric", func() {
				err := metricsWrapper.CreateWithTx(tx, policies)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreCreateWithTxError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreCreateWithTxErrorTime"))
			})
		})
	})

	Describe("All", func() {
		BeforeEach(func() {
			fakeStore.AllReturns(policies, nil)
		})
		It("returns the result of All on the Store", func() {
			returnedPolicies, err := metricsWrapper.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedPolicies).To(Equal(policies))

			Expect(fakeStore.AllCallCount()).To(Equal(1))
		})

		It("emits a metric", func() {
			_, err := metricsWrapper.All()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreAllSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.AllReturns(nil, errors.New("banana"))
			})
			It("emits an error metric", func() {
				_, err := metricsWrapper.All()
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreAllError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreAllErrorTime"))

			})
		})
	})

	Describe("ByGuids", func() {
		BeforeEach(func() {
			fakeStore.ByGuidsReturns(policies, nil)
		})
		It("returns the result of ByGuids on the Store", func() {
			returnedPolicies, err := metricsWrapper.ByGuids(srcGuids)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedPolicies).To(Equal(policies))

			Expect(fakeStore.ByGuidsCallCount()).To(Equal(1))
			returnedSrcGuids := fakeStore.ByGuidsArgsForCall(0)
			Expect(returnedSrcGuids).To(Equal(srcGuids))
		})

		It("emits a metric", func() {
			_, err := metricsWrapper.ByGuids(srcGuids)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreByGuidsSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.ByGuidsReturns(nil, errors.New("banana"))
			})
			It("emits an error metric", func() {
				_, err := metricsWrapper.ByGuids(srcGuids)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreByGuidsError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreByGuidsErrorTime"))

			})
		})
	})

	Describe("DeleteWithTx", func() {
		It("calls DeleteWithTx on the Store", func() {
			err := metricsWrapper.DeleteWithTx(tx, policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.DeleteWithTxCallCount()).To(Equal(1))
			passedTx, passedPolicies := fakeStore.DeleteWithTxArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedPolicies).To(Equal(policies))
		})

		It("emits a metric", func() {
			err := metricsWrapper.DeleteWithTx(tx, policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreDeleteWithTxSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.DeleteWithTxReturns(errors.New("banana"))
			})
			It("emits an error metric", func() {
				err := metricsWrapper.DeleteWithTx(tx, policies)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreDeleteWithTxError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreDeleteWithTxErrorTime"))
			})
		})
	})
})
