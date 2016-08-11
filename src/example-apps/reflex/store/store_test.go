package store_test

import (
	"example-apps/reflex/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var (
		addrStore          *store.Store
		stalenessThreshold int
	)

	BeforeEach(func() {
		stalenessThreshold = 10
		addrStore = store.New("local-addr", stalenessThreshold)
	})

	Describe("Add", func() {
		It("adds the addresses to the store", func() {
			addrStore.Add([]string{"some-addr", "some-other-addr"})
			Expect(addrStore.GetAddresses()).To(ConsistOf("some-addr", "some-other-addr", "local-addr"))
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
	})
})
