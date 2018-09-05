package store_test

import (
	"errors"
	"fmt"
	"policy-server/db"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"
	testhelpers "test-helpers"
	"time"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Terminal Table", func() {
	Context("when using a real db", func() {
		var (
			dbConf         dbHelper.Config
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

			realDb = db.NewConnectionPool(dbConf, 200, 200, "Terminal Table Test", "Terminal Table Test", logger)

			migrate(realDb)

			terminalsTable = &store.TerminalsTable{}

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
			It("should create a Terminal and return the ID", func() {
				id, err := terminalsTable.Create(tx)
				Expect(err).ToNot(HaveOccurred())

				Expect(id).To(Equal(int64(1)))
			})
		})

		Context("Delete", func() {
			var (
				terminalID int64
			)

			BeforeEach(func() {
				var err error
				terminalID, err = terminalsTable.Create(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(terminalID).To(Equal(int64(1)))
			})

			It("deletes the terminal", func() {
				err := terminalsTable.Delete(tx, terminalID)
				Expect(err).ToNot(HaveOccurred())

				var terminalCount int
				row := tx.QueryRow(`SELECT COUNT(id) FROM terminals WHERE id = 1`)
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

			terminalsTable = &store.TerminalsTable{}
		})

		Context("Create", func() {
			It("should return an error if the driver is not supported", func() {
				tx.DriverNameReturns("db2")

				_, err := terminalsTable.Create(tx)
				Expect(err).To(MatchError("unknown driver: db2"))
			})
		})

		Context("Delete", func() {
			It("should return the sql error", func() {
				tx.ExecReturns(nil, errors.New("broke"))

				err := terminalsTable.Delete(tx, 2)
				Expect(err).To(MatchError("broke"))
			})
		})
	})
})
