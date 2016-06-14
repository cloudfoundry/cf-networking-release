package db_test

import (
	"fmt"
	"lib/db"
	"lib/testsupport"
	"net/url"

	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetConnectionPool", func() {
	var (
		testDatabase *testsupport.TestDatabase
		dbName       string
	)

	BeforeEach(func() {
		guid, err := uuid.NewV4()
		Expect(err).NotTo(HaveOccurred())

		dbName = fmt.Sprintf("test_%x", guid[:])
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)
	})

	AfterEach(func() {
		if testDatabase != nil {
			testDatabase.Destroy()
			testDatabase = nil
		}
	})

	It("returns a database reference", func() {
		database, err := db.GetConnectionPool(testDatabase.URL())
		Expect(err).NotTo(HaveOccurred())
		defer database.Close()

		var databaseName string
		err = database.QueryRow("SELECT current_database();").Scan(&databaseName)
		Expect(err).NotTo(HaveOccurred())
		Expect(databaseName).To(Equal(dbName))
	})

	Context("when the database cannot be accessed", func() {
		It("returns a non-retryable error", func() {
			url := testDatabase.URL()

			testDatabase.Destroy()
			testDatabase = nil

			_, err := db.GetConnectionPool(url)
			Expect(err).To(HaveOccurred())

			Expect(err).NotTo(BeAssignableToTypeOf(db.RetriableError{}))
			Expect(err).To(MatchError(ContainSubstring("unable to ping")))
		})
	})

	Context("when there is a network connectivity problem", func() {
		It("returns a retriable error", func() {
			url, err := url.Parse(testDatabase.URL())
			Expect(err).NotTo(HaveOccurred())

			url.Host = "localhost:0"

			_, err = db.GetConnectionPool(url.String())
			Expect(err).To(HaveOccurred())

			Expect(err).To(BeAssignableToTypeOf(db.RetriableError{}))
			Expect(err.Error()).To(ContainSubstring("unable to ping"))
		})
	})
})
