package store_test

import (
	"errors"

	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/store/fakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecurityGroupsMetricsWrapper", func() {
	var (
		metricsWrapper    *store.SecurityGroupsMetricsWrapper
		newSecurityGroups []store.SecurityGroup
		fakeMetricsSender *fakes.MetricsSender
		fakeStore         *fakes.SecurityGroupsStore
	)

	BeforeEach(func() {
		fakeStore = &fakes.SecurityGroupsStore{}
		fakeMetricsSender = &fakes.MetricsSender{}
		metricsWrapper = &store.SecurityGroupsMetricsWrapper{
			Store:         fakeStore,
			MetricsSender: fakeMetricsSender,
		}
		newSecurityGroups = []store.SecurityGroup{
			{Guid: "some-asg-guid"},
		}
	})

	Describe("Replace", func() {
		It("calls Replace on the Store", func() {
			err := metricsWrapper.Replace(newSecurityGroups)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.ReplaceCallCount()).To(Equal(1))
			replaceArgs := fakeStore.ReplaceArgsForCall(0)
			Expect(replaceArgs).To(Equal(newSecurityGroups))
		})

		It("emits a metric", func() {
			err := metricsWrapper.Replace(newSecurityGroups)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
			name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
			Expect(name).To(Equal("SecurityGroupsStoreReplaceSuccessTime"))
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.ReplaceReturns(errors.New("banana"))
			})

			It("emits an error metric", func() {
				err := metricsWrapper.Replace(newSecurityGroups)
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("SecurityGroupsStoreReplaceError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("SecurityGroupsStoreReplaceErrorTime"))
			})
		})
	})

	Describe("BySpaceGuids", func() {
		var securityGroups []store.SecurityGroup

		Context("with space guid args", func() {
			BeforeEach(func() {
				securityGroups = []store.SecurityGroup{
					{
						Guid:              "security-group-1",
						StagingSpaceGuids: []string{"space-a"},
					},
				}
				fakeStore.BySpaceGuidsReturns(securityGroups, store.Pagination{Next: 2}, nil)
			})

			It("returns the result of BySpaceGuid on the Store", func() {
				returnedSecurityGroups, pagination, err := metricsWrapper.BySpaceGuids([]string{"space-a"}, store.Page{From: 1})
				Expect(err).NotTo(HaveOccurred())
				Expect(returnedSecurityGroups).To(Equal(securityGroups))
				Expect(pagination).To(Equal(store.Pagination{Next: 2}))

				Expect(fakeStore.BySpaceGuidsCallCount()).To(Equal(1))
				spaceGuids, page := fakeStore.BySpaceGuidsArgsForCall(0)
				Expect(spaceGuids).To(Equal([]string{"space-a"}))
				Expect(page).To(Equal(store.Page{From: 1}))
			})

			It("emits a metric", func() {
				_, _, err := metricsWrapper.BySpaceGuids([]string{"space-a"}, store.Page{From: 1})
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("SecurityGroupsStoreBySpaceGuidsSuccessTime"))
			})
		})

		Context("when there is an error", func() {
			BeforeEach(func() {
				fakeStore.BySpaceGuidsReturns(nil, store.Pagination{}, errors.New("banana"))
			})

			It("emits an error metric", func() {
				_, _, err := metricsWrapper.BySpaceGuids([]string{"space-a"}, store.Page{From: 1})
				Expect(err).To(MatchError("banana"))

				Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
				Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("SecurityGroupsStoreBySpaceGuidsError"))

				Expect(fakeMetricsSender.SendDurationCallCount()).To(Equal(1))
				name, _ := fakeMetricsSender.SendDurationArgsForCall(0)
				Expect(name).To(Equal("SecurityGroupsStoreBySpaceGuidsErrorTime"))
			})
		})
	})
})
