package store_test

import (
	"errors"
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
	)

	BeforeEach(func() {
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

	Describe("Create", func() {
		It("calls Create on the Store", func() {
			err := metricsWrapper.Create(policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.CreateCallCount()).To(Equal(1))
			passedPolicies := fakeStore.CreateArgsForCall(0)
			Expect(passedPolicies).To(Equal(policies))
		})

		It("emits a metric", func() {
			err := metricsWrapper.Create(policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreCreateSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.CreateReturns(errors.New("banana"))
			})
			It("emits an error metric", func() {
				err := metricsWrapper.Create(policies)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreCreateError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreCreateErrorTime"))
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

	Describe("Delete", func() {
		It("calls Delete on the Store", func() {
			err := metricsWrapper.Delete(policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.DeleteCallCount()).To(Equal(1))
			passedPolicies := fakeStore.DeleteArgsForCall(0)
			Expect(passedPolicies).To(Equal(policies))
		})

		It("emits a metric", func() {
			err := metricsWrapper.Delete(policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreDeleteSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.DeleteReturns(errors.New("banana"))
			})
			It("emits an error metric", func() {
				err := metricsWrapper.Delete(policies)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreDeleteError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreDeleteErrorTime"))
			})
		})
	})
})
