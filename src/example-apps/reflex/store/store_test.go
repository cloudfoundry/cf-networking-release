package store_test

import (
	"example-apps/reflex/fakes"
	"example-apps/reflex/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var (
		addrStore          *store.Store
		lock               *fakes.Mutex
		stalenessThreshold int
	)

	BeforeEach(func() {
		stalenessThreshold = 10
		lock = &fakes.Mutex{}
		addrStore = store.New("local-addr", stalenessThreshold, lock)
	})

	Describe("Add", func() {
		It("adds the addresses to the store", func() {
			addrStore.Add([]string{"some-addr", "some-other-addr"})
			Expect(addrStore.GetAddresses()).To(ConsistOf("some-addr", "some-other-addr", "local-addr"))
		})

		It("locks and unlocks the store", func() {
			addrStore.Add([]string{"some-addr", "some-other-addr"})
			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
		})
	})

	Describe("GetAddresses", func() {
		It("returns the non-stale addresses", func() {
			addrStore.Add([]string{"some-addr", "some-other-addr"})
			Expect(addrStore.GetAddresses()).To(ConsistOf("some-addr", "some-other-addr", "local-addr"))

			for i := 0; i < stalenessThreshold-1; i++ {
				addrStore.Add([]string{"some-addr"})
				Expect(addrStore.GetAddresses()).To(ConsistOf("some-addr", "some-other-addr", "local-addr"))
			}
			addrStore.Add([]string{"some-addr"})
			Expect(addrStore.GetAddresses()).To(ConsistOf("some-addr", "local-addr"))
		})

		It("locks and unlocks the store", func() {
			addrStore.Add([]string{"some-addr", "some-other-addr"})
			Expect(lock.LockCallCount()).To(Equal(1))
			Expect(lock.UnlockCallCount()).To(Equal(1))
		})
	})
})
