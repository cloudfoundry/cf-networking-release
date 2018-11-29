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
			createdPolicies := []store.EgressPolicy{{ID: "hi"}}
			fakeStore.CreateReturns(createdPolicies, nil)
			returnedPolicies, err := metricsWrapper.Create(policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.CreateCallCount()).To(Equal(1))
			passedPolicies := fakeStore.CreateArgsForCall(0)
			Expect(passedPolicies).To(Equal(policies))
			Expect(returnedPolicies).To(Equal(createdPolicies))
		})

		It("emits a metric", func() {
			_, err := metricsWrapper.Create(policies)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreCreateSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.CreateReturns(nil, errors.New("banana"))
			})

			It("emits an error metric", func() {
				_, err := metricsWrapper.Create(policies)
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

	Describe("GetBySourceGuids", func() {
		BeforeEach(func() {
			fakeStore.GetBySourceGuidsReturns(policies, nil)
		})
		It("returns the result of GetBySourceGuids on the Store", func() {
			returnedPolicies, err := metricsWrapper.GetBySourceGuids(srcGuids)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedPolicies).To(Equal(policies))

			Expect(fakeStore.GetBySourceGuidsCallCount()).To(Equal(1))
			returnedSrcGuids := fakeStore.GetBySourceGuidsArgsForCall(0)
			Expect(returnedSrcGuids).To(Equal(srcGuids))
		})

		It("emits a metric", func() {
			_, err := metricsWrapper.GetBySourceGuids(srcGuids)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreGetBySourceGuidsSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.GetBySourceGuidsReturns(nil, errors.New("banana"))
			})
			It("emits an error metric", func() {
				_, err := metricsWrapper.GetBySourceGuids(srcGuids)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreGetBySourceGuidsError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreGetBySourceGuidsErrorTime"))

			})
		})
	})

	Describe("Delete", func() {
		var egressPolicies []store.EgressPolicy
		BeforeEach(func() {
			egressPolicies = []store.EgressPolicy{{}}
			fakeStore.DeleteReturns(egressPolicies, nil)
		})

		It("calls Delete on the Store", func() {
			policies, err := metricsWrapper.Delete("some-policy-guid")
			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(Equal(egressPolicies))

			Expect(fakeStore.DeleteCallCount()).To(Equal(1))
			passedPolicies := fakeStore.DeleteArgsForCall(0)
			Expect(passedPolicies).To(ConsistOf("some-policy-guid"))
		})

		It("emits a metric", func() {
			_, err := metricsWrapper.Delete("some-policy-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreDeleteSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.DeleteReturns(nil, errors.New("banana"))
			})
			It("emits an error metric", func() {
				_, err := metricsWrapper.Delete("some-policy-guid")
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreDeleteError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreDeleteErrorTime"))
			})
		})
	})
	Describe("GetByFilter", func() {
		var sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles []string
		BeforeEach(func() {
			fakeStore.GetByFilterReturns(policies, nil)
			sourceIds = []string{"source1", "source2"}
			sourceTypes = []string{"type1", "type2"}
			destinationIds = []string{"dstid"}
			destinationNames = []string{"destName"}
			appLifecycles = []string{"#applyfe"}
		})
		It("returns the result of GetByFilter on the Store", func() {
			returnedPolicies, err := metricsWrapper.GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedPolicies).To(Equal(policies))

			Expect(fakeStore.GetByFilterCallCount()).To(Equal(1))
			actualSourceIds, actualSourceTypes, actualDestinationIds, actualDestinationNames, actualAppLifecycles := fakeStore.GetByFilterArgsForCall(0)
			Expect(sourceIds).To(Equal(actualSourceIds))
			Expect(sourceTypes).To(Equal(actualSourceTypes))
			Expect(destinationIds).To(Equal(actualDestinationIds))
			Expect(destinationNames).To(Equal(actualDestinationNames))
			Expect(appLifecycles).To(Equal(actualAppLifecycles))
		})

		It("emits a metric", func() {
			_, err := metricsWrapper.GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("EgressPolicyStoreGetByFilterSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.GetByFilterReturns(nil, errors.New("banana"))
			})
			It("emits an error metric", func() {
				_, err := metricsWrapper.GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("EgressPolicyStoreGetByFilterError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("EgressPolicyStoreGetByFilterErrorTime"))
			})
		})
	})
})
