package store_test

import (
	"errors"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DestinationMetadata", func() {
	var (
		tx                       *dbfakes.Transaction
		destinationMetadataTable *store.DestinationMetadataTable
	)

	BeforeEach(func() {
		tx = new(dbfakes.Transaction)

		destinationMetadataTable = &store.DestinationMetadataTable{}
	})

	Context("when the db fails to insert", func() {
		Context("on mysql", func() {
			BeforeEach(func() {
				tx.DriverNameReturns("mysql")
				tx.ExecReturns(nil, errors.New("failed to insert"))
			})

			It("returns an error", func() {
				_, err := destinationMetadataTable.Create(tx, "term-guid", "some-name", "some-desc")
				Expect(err).To(MatchError("failed to create destination metadata: failed to insert"))
			})
		})
	})
})
