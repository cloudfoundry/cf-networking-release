package store_test

import (
	"netman-agent/models"
	"netman-agent/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	It("supports Add, Del, and Get operations", func() {
		theStore := store.New()
		Expect(theStore.GetContainers()).To(BeEmpty())
		Expect(theStore.Add("c1", "g1", "ip1")).To(Succeed())
		Expect(theStore.Add("c2", "g2", "ip2")).To(Succeed())
		Expect(theStore.Add("c2b", "g2", "ip2b")).To(Succeed())

		firstCopy := theStore.GetContainers()
		Expect(firstCopy).To(HaveKeyWithValue("g1",
			[]models.Container{{IP: "ip1", ID: "c1"}}))
		Expect(firstCopy).To(HaveKeyWithValue("g2",
			[]models.Container{
				{IP: "ip2", ID: "c2"},
				{IP: "ip2b", ID: "c2b"},
			}))

		By("deleting an object and getting a new copy")
		Expect(theStore.Del("c1")).To(Succeed())
		secondCopy := theStore.GetContainers()
		Expect(secondCopy).NotTo(HaveKey("g1"))
		Expect(firstCopy).To(HaveKeyWithValue("g2",
			[]models.Container{
				{IP: "ip2", ID: "c2"},
				{IP: "ip2b", ID: "c2b"},
			}))

		By("deleting one more object")
		Expect(theStore.Del("c2b")).To(Succeed())
		thirdCopy := theStore.GetContainers()
		Expect(thirdCopy).To(HaveKeyWithValue("g2",
			[]models.Container{
				{IP: "ip2", ID: "c2"},
			},
		))

		By("checking that the first copy is not modified")
		Expect(firstCopy).To(HaveKeyWithValue("g1",
			[]models.Container{{IP: "ip1", ID: "c1"}}))
		Expect(firstCopy).To(HaveKeyWithValue("g2",
			[]models.Container{
				{IP: "ip2", ID: "c2"},
				{IP: "ip2b", ID: "c2b"},
			}))

	})
})
