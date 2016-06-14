package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	"lib/db"
	"lib/testsupport"
	"math/rand"
	"policy-server/fakes"
	"policy-server/models"
	"policy-server/store"

	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var dataStore store.Store
	var testDatabase *testsupport.TestDatabase
	var realDb *sqlx.DB
	var mockDb *fakes.Db

	BeforeEach(func() {
		mockDb = &fakes.Db{}

		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)

		var err error
		realDb, err = db.GetConnectionPool(testDatabase.URL())
		Expect(err).NotTo(HaveOccurred())
		dataStore, err = store.New(realDb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		if testDatabase != nil {
			testDatabase.Destroy()
		}
	})

	Describe("Connecting to the database and migrating", func() {
		Context("when the tables already exist", func() {
			It("succeeds", func() {
				_, err := store.New(realDb)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.ExecReturns(nil, errors.New("some error"))
			})

			It("should return a sensible error", func() {
				_, err := store.New(mockDb)
				Expect(err).To(MatchError("setting up tables: some error"))
			})
		})
	})

	Describe("Create", func() {
		It("saves the policies", func() {
			policies := []models.Policy{{
				Source: models.Source{"some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}, {
				Source: models.Source{"another-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "udp",
					Port:     1234,
				},
			}}

			err := dataStore.Create(policies)
			Expect(err).NotTo(HaveOccurred())

			p, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(p)).To(Equal(2))
		})

		Context("when a policy with the same content already exists", func() {
			It("does not duplicate table rows", func() {
				policies := []models.Policy{}

				err := dataStore.Create(policies)
				Expect(err).NotTo(HaveOccurred())

				policyDuplicate := []models.Policy{{
					Source: models.Source{"some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}, {
					Source: models.Source{"some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err = dataStore.Create(policyDuplicate)
				Expect(err).NotTo(HaveOccurred())

				p, err := dataStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(p)).To(Equal(1))
			})
		})

		// Context("when a Group create record fails", func() {
		// 	It("returns a error", func() {
		// 		err = dataStore.Create()
		// 		Expect(err).NotTo(HaveOccurred())
		// 	})
		// })

		// 	Context("when the db operation fails", func() {
		// 		Context("when the failure is an unexpected pq error", func() {
		// 			BeforeEach(func() {
		// 				mockDb.NamedExecReturns(nil,
		// 					&pq.Error{
		// 						Code: "2201G",
		// 					})
		// 			})

		// 			It("should return the error code", func() {
		// 				store, err := store.New(mockDb)
		// 				Expect(err).NotTo(HaveOccurred())

		// 				err = store.Create(models.Container{})
		// 				Expect(err).To(MatchError("insert: invalid_argument_for_width_bucket_function"))
		// 			})
		// 		})

		// 		Context("when the failure is not a pq Error", func() {
		// 			BeforeEach(func() {
		// 				mockDb.NamedExecReturns(nil, errors.New("some-insert-error"))
		// 			})

		// 			It("should return a sensible error", func() {
		// 				store, err := store.New(mockDb)
		// 				Expect(err).NotTo(HaveOccurred())

		// 				err = store.Create(models.Container{})
		// 				Expect(err).To(MatchError("insert: some-insert-error"))
		// 			})
		// 		})
		// 	})
	})

	Describe("All", func() {
		var expectedPolicies []models.Policy

		BeforeEach(func() {
			expectedPolicies = []models.Policy{models.Policy{
				Source: models.Source{"some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}}

			err := dataStore.Create(expectedPolicies)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all containers that have been added", func() {
			policies, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(ConsistOf(expectedPolicies))
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.QueryReturns(nil, errors.New("some query error"))
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.All()
				Expect(err).To(MatchError("listing all: some query error"))
			})
		})

		Context("when the query result parsing fails", func() {
			var rows *sql.Rows

			BeforeEach(func() {
				expectedPolicies = []models.Policy{models.Policy{
					Source: models.Source{"some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err := dataStore.Create(expectedPolicies)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.New(realDb)
				Expect(err).NotTo(HaveOccurred())
				rows, err = realDb.Query(`select * from policies`)
				Expect(err).NotTo(HaveOccurred())

				mockDb.QueryReturns(rows, nil)
			})

			AfterEach(func() {
				rows.Close()
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.All()
				Expect(err).To(MatchError("listing all: sql: expected 3 destination arguments in Scan, not 4"))
			})
		})
	})
})
