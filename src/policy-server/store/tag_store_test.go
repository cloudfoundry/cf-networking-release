package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	dbFakes "policy-server/db/fakes"
	"policy-server/store"
	"policy-server/store/fakes"
	"time"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	"policy-server/db"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagStore", func() {
	var (
		dataStore   store.Store
		dbConf      dbHelper.Config
		realDb      *db.ConnWrapper
		mockDb      *fakes.Db
		group       store.GroupRepo
		destination store.DestinationRepo
		policy      store.PolicyRepo

		tagStore  store.TagStore
		tagLength int
	)

	BeforeEach(func() {

		tagLength = 1
		mockDb = &fakes.Db{}

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("tag_store_test_node_%d", time.Now().UnixNano())

		testsupport.CreateDatabase(dbConf)

		logger := lager.NewLogger("Tag Store Test")

		var err error
		realDb = db.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Tag Store Test", "Tag Store Test", logger)
		Expect(err).NotTo(HaveOccurred())

		group = &store.GroupTable{}
		destination = &store.DestinationTable{}
		policy = &store.PolicyTable{}

		mockDb.DriverNameReturns(realDb.DriverName())

		migrateAndPopulateTags(realDb, tagLength)
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testsupport.RemoveDatabase(dbConf)
	})

	Describe("CreateTag", func() {
		var (
			groupGuid string
			groupType string
		)

		BeforeEach(func() {
			tagStore = store.NewTagStore(realDb, group, tagLength)
			groupGuid, groupType = "meow-guid", "meow-type"
		})

		It("saves the group", func() {
			tag, err := tagStore.CreateTag(groupGuid, groupType)
			Expect(err).NotTo(HaveOccurred())
			Expect(tag).To(Equal(store.Tag{ID: "meow-guid", Type: "meow-type", Tag: "01"}))

			t, err := tagStore.Tags()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(t)).To(Equal(1))
		})

		Context("when a group with the same type and guid exists", func() {
			var expectedTag store.Tag

			BeforeEach(func() {
				var err error
				expectedTag, err = tagStore.CreateTag(groupGuid, groupType)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should get the same tag", func() {
				tag, err := tagStore.CreateTag(groupGuid, groupType)
				Expect(err).NotTo(HaveOccurred())
				Expect(tag).To(Equal(expectedTag))

				t, err := tagStore.Tags()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(t)).To(Equal(1))
			})
		})

		Context("when there are no tags left to allocate", func() {
			var (
				mockTx    *dbFakes.Transaction
				mockGroup *fakes.GroupRepo
			)

			BeforeEach(func() {
				mockGroup = &fakes.GroupRepo{}
				mockGroup.CreateReturns(-1, errors.New("failed to find available tag"))
				mockTx = &dbFakes.Transaction{}
				mockDb.BeginxReturns(mockTx, nil)

				tagStore = store.NewTagStore(mockDb, mockGroup, tagLength)
			})

			It("returns an error", func() {
				_, err := tagStore.CreateTag(groupGuid, groupType)
				Expect(err).To(MatchError(ContainSubstring("failed to find available tag")))
			})

			It("rolls back the transaction", func() {
				tagStore.CreateTag(groupGuid, groupType)
				Expect(mockTx.RollbackCallCount()).To(Equal(1))
			})
		})

		Context("when a transaction commit fails", func() {
			var (
				mockTx    *dbFakes.Transaction
				mockGroup *fakes.GroupRepo
			)

			BeforeEach(func() {
				mockGroup = &fakes.GroupRepo{}
				mockGroup.CreateReturns(1, nil)
				mockTx = &dbFakes.Transaction{}
				mockTx.CommitReturns(errors.New("transaction commit failed"))
				mockDb.BeginxReturns(mockTx, nil)

				tagStore = store.NewTagStore(mockDb, mockGroup, tagLength)
			})

			It("returns an error", func() {
				_, err := tagStore.CreateTag(groupGuid, groupType)
				Expect(err).To(MatchError(ContainSubstring("transaction commit failed")))
			})

			It("rolls back the transaction", func() {
				tagStore.CreateTag(groupGuid, groupType)
				Expect(mockTx.RollbackCallCount()).To(Equal(1))
			})
		})
	})

	Describe("Tags", func() {
		BeforeEach(func() {
			tagStore = store.NewTagStore(realDb, group, tagLength)
			dataStore = store.New(realDb, group, destination, policy, 1)
		})

		BeforeEach(func() {
			policies := []store.Policy{{
				Source: store.Source{ID: "some-app-guid"},
				Destination: store.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}, {
				Source: store.Source{ID: "some-app-guid"},
				Destination: store.Destination{
					ID:       "another-app-guid",
					Protocol: "udp",
					Port:     5555,
				},
			}}

			err := dataStore.Create(policies)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all tags that have been added", func() {
			tags, err := tagStore.Tags()
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(ConsistOf([]store.Tag{
				{ID: "some-app-guid", Tag: "01", Type: "app"},
				{ID: "some-other-app-guid", Tag: "02", Type: "app"},
				{ID: "another-app-guid", Tag: "03", Type: "app"},
			}))
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.QueryReturns(nil, errors.New("some query error"))
			})

			It("should return a sensible error", func() {
				store := store.NewTagStore(mockDb, group, tagLength)

				_, err := store.Tags()
				Expect(err).To(MatchError("listing tags: some query error"))
			})
		})

		Context("when the query result parsing fails", func() {
			var rows *sql.Rows

			BeforeEach(func() {
				var err error
				rows, err = realDb.Query(`select id from groups`)
				Expect(err).NotTo(HaveOccurred())

				mockDb.QueryReturns(rows, nil)
			})

			AfterEach(func() {
				rows.Close()
			})

			It("should return a sensible error", func() {
				store := store.NewTagStore(mockDb, group, tagLength)

				_, err := store.Tags()
				Expect(err).To(MatchError(ContainSubstring("listing tags: sql: expected")))
			})
		})
	})
})
