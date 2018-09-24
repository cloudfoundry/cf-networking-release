package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	"policy-server/store"
	"policy-server/store/fakes"
	"strings"
	"test-helpers"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tag Populator", func() {
	var (
		tagPopulator *store.TagPopulator
	)

	Context("when connecting to the DB succeeds", func() {
		var (
			dbConf db.Config
			realDb *db.ConnWrapper
		)

		BeforeEach(func() {

			dbConf = testsupport.GetDBConfig()
			dbConf.DatabaseName = fmt.Sprintf("tag_populator_test_node_%d", time.Now().UnixNano())

			testhelpers.CreateDatabase(dbConf)

			logger := lager.NewLogger("Tag Populator Test")

			var err error
			realDb, err = db.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Tag Populator Test", "Tag Populator Test", logger)
			Expect(err).NotTo(HaveOccurred())

			migrate(realDb)
			tagPopulator = &store.TagPopulator{
				DBConnection: realDb,
			}
		})

		AfterEach(func() {
			if realDb != nil {
				Expect(realDb.Close()).To(Succeed())
			}
			testhelpers.RemoveDatabase(dbConf)
		})

		Context("when the groups table is being populated", func() {
			It("does not exceed 2^(tag_length * 8) rows", func() {
				tagPopulator.PopulateTables(1)
				var id int
				err := realDb.QueryRow(`SELECT id FROM groups ORDER BY id DESC LIMIT 1`).Scan(&id)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(255))
			})
		})

		Context("when the groups table is ALREADY populated", func() {
			It("does not add more rows", func() {
				tagPopulator.PopulateTables(1)
				var id int
				err := realDb.QueryRow(`SELECT id FROM groups ORDER BY id DESC LIMIT 1`).Scan(&id)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(255))

				tagPopulator.PopulateTables(2)
				Expect(err).NotTo(HaveOccurred())

				err = realDb.QueryRow(`SELECT id FROM groups ORDER BY id DESC LIMIT 1`).Scan(&id)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(255))
			})
		})
	})

	Context("when the groups table fails to populate", func() {
		var (
			mockDb *fakes.Db
		)

		BeforeEach(func() {
			mockDb = &fakes.Db{}

			mockDb.ExecStub = func(sql string, t ...interface{}) (sql.Result, error) {
				if strings.Contains(sql, "INSERT") {
					return nil, errors.New("some error")
				}
				return nil, nil
			}

			tagPopulator = &store.TagPopulator{
				DBConnection: mockDb,
			}
		})

		It("returns an error", func() {
			err := tagPopulator.PopulateTables(1)
			Expect(err).To(MatchError("populating tables: some error"))
		})
	})
})
