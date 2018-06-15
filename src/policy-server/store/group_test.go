package store_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"policy-server/store"
	"policy-server/db/fakes"
)

var _ = Describe("Group Repo", func() {

	var (
		gt store.GroupTable
		tx *fakes.Transaction
	)

	BeforeEach(func(){
		tx = &fakes.Transaction{}

	})

	Describe("Create", func(){
		BeforeEach(func(){
		})

		It("creates a group", func(){
			gt = store.GroupTable{}
			id, err := gt.Create(tx, "guid", "type")
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(1))

			Expect(tx.QueryRowCallCount()).To(Equal(2))

		})
		Context("When a group with provided guid and type already exists",func(){})
		Context("When a group with provided guid already exists, but with a different type",func(){})
		Context("When there are no tags available", func(){})
	})
})
