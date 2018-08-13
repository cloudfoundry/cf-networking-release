package store_test

import (
	"errors"
	"policy-server/store"
	"policy-server/store/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyCollectionMetricsWrapper", func() {
	var (
		metricsWrapper   *store.PolicyCollectionMetricsWrapper
		collectionStore  *fakes.PolicyCollectionStore
		metricsSender    *fakes.MetricsSender
		policyCollection store.PolicyCollection
	)

	BeforeEach(func() {
		collectionStore = &fakes.PolicyCollectionStore{}
		metricsSender = &fakes.MetricsSender{}

		metricsWrapper = &store.PolicyCollectionMetricsWrapper{
			Store:         collectionStore,
			MetricsSender: metricsSender,
		}

		policyCollection = store.PolicyCollection{}
	})

	Describe("Create", func() {
		It("should call create on PolicyCollectionStore", func() {
			err := metricsWrapper.Create(policyCollection)
			Expect(err).NotTo(HaveOccurred())

			Expect(collectionStore.CreateCallCount()).To(Equal(1))
			Expect(collectionStore.CreateArgsForCall(0)).To(Equal(policyCollection))
		})

		It("should emit metrics", func() {
			err := metricsWrapper.Create(policyCollection)
			Expect(err).NotTo(HaveOccurred())

			Expect(metricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := metricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("StoreCreateSuccessTime"))
		})

		It("should emit error metrics when create fails", func() {
			expectedErr := errors.New("oh no it failed, how sad")
			collectionStore.CreateReturns(expectedErr)

			err := metricsWrapper.Create(policyCollection)
			Expect(err).To(Equal(expectedErr))

			Expect(metricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(metricsSender.IncrementCounterArgsForCall(0)).To(Equal("StoreCreateError"))
			Expect(metricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := metricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("StoreCreateErrorTime"))
		})
	})

	Describe("Delete", func() {
		It("should call delete on PolicyCollectionStore", func() {
			err := metricsWrapper.Delete(policyCollection)
			Expect(err).NotTo(HaveOccurred())

			Expect(collectionStore.DeleteCallCount()).To(Equal(1))
			Expect(collectionStore.DeleteArgsForCall(0)).To(Equal(policyCollection))
		})

		It("should emit metrics", func() {
			err := metricsWrapper.Delete(policyCollection)
			Expect(err).NotTo(HaveOccurred())

			Expect(metricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := metricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("StoreDeleteSuccessTime"))
		})

		It("should emit error metrics when delete fails", func() {
			expectedErr := errors.New("oh no it failed, how sad")
			collectionStore.DeleteReturns(expectedErr)

			err := metricsWrapper.Delete(policyCollection)
			Expect(err).To(Equal(expectedErr))

			Expect(metricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(metricsSender.IncrementCounterArgsForCall(0)).To(Equal("StoreDeleteError"))
			Expect(metricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := metricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("StoreDeleteErrorTime"))
		})
	})

	Describe("All", func() {
		It("should call all on PolicyCollectionStore", func() {
			collectionStore.AllReturns(policyCollection, nil)
			returnedPolicies, err := metricsWrapper.All()
			Expect(err).NotTo(HaveOccurred())

			Expect(collectionStore.AllCallCount()).To(Equal(1))
			Expect(returnedPolicies).To(Equal(policyCollection))
		})

		It("should emit metrics", func() {
			_, err := metricsWrapper.All()
			Expect(err).NotTo(HaveOccurred())

			Expect(metricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := metricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("StoreAllSuccessTime"))
		})

		It("should emit error metrics when delete fails", func() {
			expectedErr := errors.New("oh no it failed, how sad")
			collectionStore.AllReturns(store.PolicyCollection{}, expectedErr)

			_, err := metricsWrapper.All()
			Expect(err).To(Equal(expectedErr))

			Expect(metricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(metricsSender.IncrementCounterArgsForCall(0)).To(Equal("StoreAllError"))
			Expect(metricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := metricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("StoreAllErrorTime"))
		})
	})
})
