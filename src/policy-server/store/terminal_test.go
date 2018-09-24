package store_test

import (
	"errors"
	"fmt"
	"policy-server/store"
	testhelpers "test-helpers"
	"time"

	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Terminal Table", func() {
	Context("when using a real db", func() {
		var (
			dbConf         db.Config
			realDb         *db.ConnWrapper
			terminalsTable *store.TerminalsTable
			tx             db.Transaction
		)

		BeforeEach(func() {
			var err error
			dbConf = testsupport.GetDBConfig()
			dbConf.DatabaseName = fmt.Sprintf("terminal_table_test_node_%d", time.Now().UnixNano())
			dbConf.Timeout = 30
			testhelpers.CreateDatabase(dbConf)

			logger := lager.NewLogger("Terminal Table Test")

			realDb, err = db.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Terminal Table Test", "Terminal Table Test", logger)
			Expect(err).NotTo(HaveOccurred())

			migrate(realDb)

			terminalsTable = &store.TerminalsTable{
				Guids: &store.GuidGenerator{},
			}

			tx, err = realDb.Beginx()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			tx.Rollback()
			if realDb != nil {
				Expect(realDb.Close()).To(Succeed())
			}
			testhelpers.RemoveDatabase(dbConf)
		})

		Context("Create", func() {
			It("should create a Terminal and return the guid", func() {
				guid, err := terminalsTable.Create(tx)
				Expect(err).ToNot(HaveOccurred())
				_, err = uuid.ParseHex(guid)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Delete", func() {
			var (
				terminalID string
			)

			BeforeEach(func() {
				var err error
				terminalID, err = terminalsTable.Create(tx)
				Expect(err).ToNot(HaveOccurred())
			})

			It("deletes the terminal", func() {
				err := terminalsTable.Delete(tx, terminalID)
				Expect(err).ToNot(HaveOccurred())

				var terminalCount int
				row := tx.QueryRow(tx.Rebind(`SELECT COUNT(guid) FROM terminals WHERE guid = ?`), terminalID)
				err = row.Scan(&terminalCount)
				Expect(err).ToNot(HaveOccurred())
				Expect(terminalCount).To(Equal(0))
			})
		})
	})

	Context("database error cases", func() {
		var (
			tx *dbfakes.Transaction

			terminalsTable *store.TerminalsTable
		)

		BeforeEach(func() {
			tx = &dbfakes.Transaction{}

			terminalsTable = &store.TerminalsTable{
				Guids: &store.GuidGenerator{},
			}
		})

		Context("Create", func() {
			It("should return the sql error", func() {
				tx.ExecReturns(nil, errors.New("broke"))

				_, err := terminalsTable.Create(tx)
				Expect(err).To(MatchError("broke"))
			})
		})

		Context("Delete", func() {
			It("should return the sql error", func() {
				tx.ExecReturns(nil, errors.New("broke"))

				err := terminalsTable.Delete(tx, "foo")
				Expect(err).To(MatchError("broke"))
			})
		})
	})
})
