package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"policy-server/models"
	"policy-server/store"
	"policy-server/store/fakes"
	"strings"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/go-db-helpers/db"
	"code.cloudfoundry.org/go-db-helpers/testsupport"

	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var dataStore store.Store
	var testDatabase *testsupport.TestDatabase
	var realDb *sqlx.DB
	var mockDb *fakes.Db
	var group store.GroupRepo
	var destination store.DestinationRepo
	var policy store.PolicyRepo

	BeforeEach(func() {
		mockDb = &fakes.Db{}

		dbName := fmt.Sprintf("test_netman_database_%x", rand.Int())
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)

		var err error
		realDb, err = db.GetConnectionPool(testDatabase.DBConfig())
		Expect(err).NotTo(HaveOccurred())

		group = &store.Group{}
		destination = &store.Destination{}
		policy = &store.Policy{}

		mockDb.DriverNameReturns(realDb.DriverName())
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		if testDatabase != nil {
			testDatabase.Destroy()
		}
	})

	Describe("concurrent create and delete requests", func() {
		It("remains consistent", func() {
			dataStore, err := store.New(realDb, group, destination, policy, 2)
			Expect(err).NotTo(HaveOccurred())

			nPolicies := 1000
			policies := []interface{}{}
			for i := 0; i < nPolicies; i++ {
				appName := fmt.Sprintf("some-app-%x", i)
				policies = append(policies, models.Policy{
					Source:      models.Source{ID: appName},
					Destination: models.Destination{ID: appName, Protocol: "tcp", Port: 1234},
				})
			}

			parallelRunner := &testsupport.ParallelRunner{
				NumWorkers: 4,
			}
			toDelete := make(chan (interface{}), nPolicies)

			go func() {
				parallelRunner.RunOnSlice(policies, func(policy interface{}) {
					p := policy.(models.Policy)
					Expect(dataStore.Create(context.Background(), []models.Policy{p})).To(Succeed())
					toDelete <- p
				})
				close(toDelete)
			}()

			var nDeleted int32
			parallelRunner.RunOnChannel(toDelete, func(policy interface{}) {
				p := policy.(models.Policy)
				Expect(dataStore.Delete(context.Background(), []models.Policy{p})).To(Succeed())
				atomic.AddInt32(&nDeleted, 1)
			})

			Expect(nDeleted).To(Equal(int32(nPolicies)))

			allPolicies, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())

			Expect(allPolicies).To(BeEmpty())
		})
	})

	Describe("New", func() {
		BeforeEach(func() {
			var err error
			dataStore, err = store.New(realDb, group, destination, policy, 1)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Connecting to the database and migrating", func() {
			Context("when the tables already exist", func() {
				It("succeeds", func() {
					_, err := store.New(realDb, group, destination, policy, 2)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the db operation fails", func() {
				BeforeEach(func() {
					mockDb.ExecReturns(nil, errors.New("some error"))
				})

				It("should return a sensible error", func() {
					_, err := store.New(mockDb, group, destination, policy, 2)
					Expect(err).To(MatchError("setting up tables: some error"))
				})
			})
		})

		Context("when the groups table is ALREADY populated", func() {
			It("does not add more rows", func() {
				var id int
				err := realDb.QueryRow(`SELECT id FROM groups ORDER BY id DESC LIMIT 1`).Scan(&id)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(255))

				_, err = store.New(realDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())

				err = realDb.QueryRow(`SELECT id FROM groups ORDER BY id DESC LIMIT 1`).Scan(&id)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(255))
			})
		})

		Context("when the groups table is being populated", func() {
			It("does not exceed 2^(tag_length * 8) rows", func() {
				var id int
				err := realDb.QueryRow(`SELECT id FROM groups ORDER BY id DESC LIMIT 1`).Scan(&id)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(255))
			})
		})

		Context("when the store is instantiated with tag length > 3", func() {
			It("returns an error", func() {
				_, err := store.New(realDb, group, destination, policy, 4)
				Expect(err).To(MatchError("tag length out of range (1-3): 4"))
			})
		})

		Context("when the store is instantiated with tag length < 1", func() {
			It("returns an error", func() {
				_, err := store.New(realDb, group, destination, policy, 0)
				Expect(err).To(MatchError("tag length out of range (1-3): 0"))
			})
		})

		Context("when the groups table fails to populate", func() {
			BeforeEach(func() {
				mockDb.ExecStub = func(sql string, t ...interface{}) (sql.Result, error) {
					if strings.Contains(sql, "INSERT") {
						return nil, errors.New("some error")
					}
					return nil, nil
				}
			})

			It("returns an error", func() {
				_, err := store.New(mockDb, group, destination, policy, 1)
				Expect(err).To(MatchError("populating tables: some error"))
			})
		})
	})

	Describe("Create", func() {
		BeforeEach(func() {
			var err error
			dataStore, err = store.New(realDb, group, destination, policy, 1)
			Expect(err).NotTo(HaveOccurred())
		})

		It("saves the policies", func() {
			policies := []models.Policy{{
				Source: models.Source{ID: "some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}, {
				Source: models.Source{ID: "another-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "udp",
					Port:     1234,
				},
			}}

			err := dataStore.Create(context.Background(), policies)
			Expect(err).NotTo(HaveOccurred())

			p, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(p)).To(Equal(2))
		})

		Context("when a policy with the same content already exists", func() {
			It("does not duplicate table rows", func() {
				policies := []models.Policy{}

				err := dataStore.Create(context.Background(), policies)
				Expect(err).NotTo(HaveOccurred())

				policyDuplicate := []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}, {
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err = dataStore.Create(context.Background(), policyDuplicate)
				Expect(err).NotTo(HaveOccurred())

				p, err := dataStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(p)).To(Equal(1))
			})
		})

		Context("when there are no tags left to allocate", func() {
			BeforeEach(func() {
				policies := []models.Policy{}
				for i := 1; i < 256; i++ {
					policies = append(policies, models.Policy{
						Source: models.Source{ID: fmt.Sprintf("%d", i)},
						Destination: models.Destination{
							ID:       fmt.Sprintf("%d", i),
							Protocol: "tcp",
							Port:     8080,
						},
					})
				}
				err := dataStore.Create(context.Background(), policies)
				Expect(err).NotTo(HaveOccurred())
				Expect(dataStore.All()).To(HaveLen(255))
			})
			It("returns an error", func() {
				policies := []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err := dataStore.Create(context.Background(), policies)
				Expect(err).To(MatchError(ContainSubstring("failed to find available tag")))
			})
		})

		Context("when a tag is freed by delete", func() {
			It("reuses the tag", func() {
				policies := []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}, {
					Source: models.Source{ID: "another-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "udp",
						Port:     1234,
					},
				}}

				err := dataStore.Create(context.Background(), policies)
				Expect(err).NotTo(HaveOccurred())

				tags, err := dataStore.Tags()
				Expect(err).NotTo(HaveOccurred())
				Expect(tags).To(ConsistOf([]models.Tag{
					{ID: "some-app-guid", Tag: "01"},
					{ID: "some-other-app-guid", Tag: "02"},
					{ID: "another-app-guid", Tag: "03"},
				}))

				err = dataStore.Delete(context.Background(), policies[:1])
				Expect(err).NotTo(HaveOccurred())

				err = dataStore.Create(context.Background(), []models.Policy{{
					Source: models.Source{ID: "yet-another-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})
				Expect(err).NotTo(HaveOccurred())

				tags, err = dataStore.Tags()
				Expect(err).NotTo(HaveOccurred())
				Expect(tags).To(ConsistOf([]models.Tag{
					{ID: "yet-another-app-guid", Tag: "01"},
					{ID: "some-other-app-guid", Tag: "02"},
					{ID: "another-app-guid", Tag: "03"},
				}))
			})
		})

		Context("when a transaction create fails", func() {
			var err error

			BeforeEach(func() {
				mockDb.BeginxReturns(nil, errors.New("some-db-error"))
				dataStore, err = store.New(mockDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err = dataStore.Create(context.Background(), nil)
				Expect(err).To(MatchError("begin transaction: some-db-error"))
			})
		})

		Context("when a Group create record fails", func() {
			var fakeGroup *fakes.GroupRepo
			var err error

			BeforeEach(func() {
				fakeGroup = &fakes.GroupRepo{}
				fakeGroup.CreateReturns(-1, errors.New("some-insert-error"))

				dataStore, err = store.New(realDb, fakeGroup, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a error", func() {
				err = dataStore.Create(context.Background(), []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})
				Expect(err).To(MatchError("creating group: some-insert-error"))
			})

		})

		Context("when the second create group fails", func() {
			var fakeGroup *fakes.GroupRepo
			var err error

			BeforeEach(func() {
				fakeGroup = &fakes.GroupRepo{}
				type response struct {
					Id  int
					Err error
				}

				responses := []response{
					{2, nil},
					{-1, errors.New("some-insert-error")},
				}
				fakeGroup.CreateStub = func(t store.Transaction, guid string) (int, error) {
					response := responses[0]
					responses = responses[1:]
					return response.Id, response.Err
				}

				dataStore, err = store.New(realDb, fakeGroup, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				err = dataStore.Create(context.Background(), []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})

				Expect(err).To(MatchError("creating group: some-insert-error"))
			})
		})

		Context("when a Destination create record fails", func() {
			var fakeDestination *fakes.DestinationRepo
			var err error

			BeforeEach(func() {
				fakeDestination = &fakes.DestinationRepo{}
				fakeDestination.CreateReturns(-1, errors.New("some-insert-error"))

				dataStore, err = store.New(realDb, group, fakeDestination, policy, 2)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a error", func() {
				err = dataStore.Create(context.Background(), []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})
				Expect(err).To(MatchError("creating destination: some-insert-error"))
				var groupsCount int
				err = realDb.QueryRow(`SELECT count(*) FROM groups WHERE guid IS NOT NULL`).Scan(&groupsCount)
				Expect(groupsCount).To(BeZero())
			})
		})

		Context("when a Policy create record fails", func() {
			var fakePolicy *fakes.PolicyRepo
			var err error

			BeforeEach(func() {
				fakePolicy = &fakes.PolicyRepo{}
				fakePolicy.CreateReturns(errors.New("some-insert-error"))

				dataStore, err = store.New(realDb, group, destination, fakePolicy, 2)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a error", func() {
				err = dataStore.Create(context.Background(), []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})
				Expect(err).To(MatchError("creating policy: some-insert-error"))
			})
		})

		Context("when the context gets cancelled", func() {
			It("returns a error", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				cancel()
				err := dataStore.Create(ctx, []models.Policy{{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})
				Expect(err).To(MatchError("context done"))
			})
		})

	})

	Describe("All", func() {
		var expectedPolicies []models.Policy

		BeforeEach(func() {
			var err error
			expectedPolicies = []models.Policy{models.Policy{
				Source: models.Source{ID: "some-app-guid", Tag: "01"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Tag:      "02",
					Protocol: "tcp",
					Port:     8080,
				},
			}}

			dataStore, err = store.New(realDb, group, destination, policy, 1)
			Expect(err).NotTo(HaveOccurred())

			err = dataStore.Create(context.Background(), expectedPolicies)
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
				store, err := store.New(mockDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.All()
				Expect(err).To(MatchError("listing all: some query error"))
			})
		})

		Context("when the query result parsing fails", func() {
			var rows *sql.Rows

			BeforeEach(func() {
				expectedPolicies = []models.Policy{models.Policy{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err := dataStore.Create(context.Background(), expectedPolicies)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.New(realDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())
				rows, err = realDb.Query(`select * from policies`)
				Expect(err).NotTo(HaveOccurred())

				mockDb.QueryReturns(rows, nil)
			})

			AfterEach(func() {
				rows.Close()
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.All()
				Expect(err).To(MatchError(ContainSubstring("listing all: sql: expected")))
			})
		})
	})

	Describe("ByGuids", func() {
		var allPolicies []models.Policy
		var expectedPolicies []models.Policy
		var err error

		BeforeEach(func() {
			allPolicies = []models.Policy{
				models.Policy{
					Source: models.Source{
						ID:  "app-guid-00",
						Tag: "01",
					},
					Destination: models.Destination{
						ID:       "app-guid-01",
						Tag:      "02",
						Protocol: "tcp",
						Port:     101,
					},
				},
				models.Policy{
					Source: models.Source{
						ID:  "app-guid-01",
						Tag: "02",
					},
					Destination: models.Destination{
						ID:       "app-guid-02",
						Tag:      "03",
						Protocol: "tcp",
						Port:     102,
					},
				},
				models.Policy{
					Source: models.Source{
						ID:  "app-guid-02",
						Tag: "03",
					},
					Destination: models.Destination{
						ID:       "app-guid-00",
						Tag:      "01",
						Protocol: "tcp",
						Port:     100,
					},
				},
				models.Policy{
					Source: models.Source{
						ID:  "app-guid-03",
						Tag: "04",
					},
					Destination: models.Destination{
						ID:       "app-guid-03",
						Tag:      "04",
						Protocol: "tcp",
						Port:     103,
					},
				},
			}

			dataStore, err = store.New(realDb, group, destination, policy, 1)
			Expect(err).NotTo(HaveOccurred())

			err = dataStore.Create(context.Background(), allPolicies)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when empty args is provided", func() {
			BeforeEach(func() {
				dataStore, err = store.New(mockDb, group, destination, policy, 1)
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns an empty slice ", func() {
				policies, err := dataStore.ByGuids(nil, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(BeEmpty())

				By("not making any queries")
				Expect(mockDb.QueryCallCount()).To(Equal(0))
			})
		})

		Context("when srcGuids is provided", func() {
			It("returns policies whose source is in srcGuids", func() {
				policies, err := dataStore.ByGuids([]string{"app-guid-00", "app-guid-01"}, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(allPolicies[0], allPolicies[1]))
			})
		})
		Context("when destGuids is provided", func() {
			It("returns policies whose destination is in destGuids", func() {
				policies, err := dataStore.ByGuids(nil, []string{"app-guid-00", "app-guid-01"})
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(allPolicies[0], allPolicies[2]))
			})
		})
		Context("when srcGuids and destGuids are provided", func() {
			It("returns policies that satisfy either srcGuids or destGuids", func() {
				policies, err := dataStore.ByGuids(
					[]string{"app-guid-00", "app-guid-01"},
					[]string{"app-guid-00", "app-guid-01"},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(
					allPolicies[0], allPolicies[1], allPolicies[2],
				))
			})
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.QueryReturns(nil, errors.New("some query error"))
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.ByGuids(
					[]string{"does-not-matter"},
					[]string{"does-not-matter"},
				)
				Expect(err).To(MatchError("listing all: some query error"))
			})
		})

		Context("when the query result parsing fails", func() {
			var rows *sql.Rows

			BeforeEach(func() {
				expectedPolicies = []models.Policy{models.Policy{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err := dataStore.Create(context.Background(), expectedPolicies)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.New(realDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())
				rows, err = realDb.Query(`select * from policies`)
				Expect(err).NotTo(HaveOccurred())

				mockDb.QueryReturns(rows, nil)
			})

			AfterEach(func() {
				rows.Close()
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.ByGuids(
					[]string{"does-not-matter"},
					[]string{"does-not-matter"},
				)
				Expect(err).To(MatchError(ContainSubstring("listing all: sql: expected")))
			})
		})
	})

	Describe("Tags", func() {
		BeforeEach(func() {
			var err error
			dataStore, err = store.New(realDb, group, destination, policy, 1)
			Expect(err).NotTo(HaveOccurred())
		})

		BeforeEach(func() {
			policies := []models.Policy{{
				Source: models.Source{ID: "some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}, {
				Source: models.Source{ID: "some-app-guid"},
				Destination: models.Destination{
					ID:       "another-app-guid",
					Protocol: "udp",
					Port:     5555,
				},
			}}

			err := dataStore.Create(context.Background(), policies)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all tags that have been added", func() {
			tags, err := dataStore.Tags()
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(ConsistOf([]models.Tag{
				{ID: "some-app-guid", Tag: "01"},
				{ID: "some-other-app-guid", Tag: "02"},
				{ID: "another-app-guid", Tag: "03"},
			}))
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.QueryReturns(nil, errors.New("some query error"))
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Tags()
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
				store, err := store.New(mockDb, group, destination, policy, 2)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Tags()
				Expect(err).To(MatchError(ContainSubstring("listing tags: sql: expected")))
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			var err error
			dataStore, err = store.New(realDb, group, destination, policy, 1)
			Expect(err).NotTo(HaveOccurred())

			err = dataStore.Create(context.Background(), []models.Policy{
				{
					Source: models.Source{ID: "some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				},
				{
					Source: models.Source{ID: "another-app-guid"},
					Destination: models.Destination{
						ID:       "yet-another-app-guid",
						Protocol: "udp",
						Port:     5555,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes the specified policies", func() {
			err := dataStore.Delete(context.Background(), []models.Policy{{
				Source: models.Source{ID: "some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			policies, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(Equal([]models.Policy{{
				Source: models.Source{ID: "another-app-guid", Tag: "03"},
				Destination: models.Destination{
					ID:       "yet-another-app-guid",
					Protocol: "udp",
					Port:     5555,
					Tag:      "04",
				},
			}}))
		})

		It("deletes the tags if no longer referenced", func() {
			err := dataStore.Delete(context.Background(), []models.Policy{{
				Source: models.Source{ID: "some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			policies, err := dataStore.Tags()
			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(Equal([]models.Tag{{
				ID:  "another-app-guid",
				Tag: "03",
			}, {
				ID:  "yet-another-app-guid",
				Tag: "04",
			}}))
		})

		Context("when an error occurs", func() {
			var fakeGroup *fakes.GroupRepo
			var fakeDestination *fakes.DestinationRepo
			var fakePolicy *fakes.PolicyRepo
			var err error

			BeforeEach(func() {
				fakeGroup = &fakes.GroupRepo{}
				fakeDestination = &fakes.DestinationRepo{}
				fakePolicy = &fakes.PolicyRepo{}
				dataStore, err = store.New(realDb, fakeGroup, fakeDestination, fakePolicy, 2)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when a transaction create fails", func() {
				var err error

				BeforeEach(func() {
					mockDb.BeginxReturns(nil, errors.New("some-db-error"))
					dataStore, err = store.New(mockDb, group, destination, policy, 2)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					err = dataStore.Delete(context.Background(), nil)
					Expect(err).To(MatchError("begin transaction: some-db-error"))
				})
			})

			Context("when getting the source id fails", func() {
				Context("when the error is because the source does not exist", func() {
					BeforeEach(func() {
						fakeGroup.GetIDStub = func(store.Transaction, string) (int, error) {
							if fakeGroup.GetIDCallCount() == 1 {
								return -1, sql.ErrNoRows
							}
							return 0, nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{
							models.Policy{Source: models.Source{ID: "0"}},
							models.Policy{Source: models.Source{ID: "apple"}, Destination: models.Destination{ID: "banana"}},
						})
						Expect(err).NotTo(HaveOccurred())
						Expect(fakeGroup.GetIDCallCount()).To(Equal(3))

						_, call0SourceID := fakeGroup.GetIDArgsForCall(0)
						Expect(call0SourceID).To(Equal("0"))

						_, call1SourceID := fakeGroup.GetIDArgsForCall(1)
						Expect(call1SourceID).To(Equal("apple"))

						_, call2SourceID := fakeGroup.GetIDArgsForCall(2)
						Expect(call2SourceID).To(Equal("banana"))
					})
				})

				Context("when the error is for any other reason", func() {
					BeforeEach(func() {
						fakeGroup.GetIDReturns(-1, errors.New("some-get-error"))
					})
					It("returns the error", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{{
							Source: models.Source{ID: "some-app-guid"},
							Destination: models.Destination{
								ID:       "some-other-app-guid",
								Protocol: "tcp",
								Port:     8080,
							},
						}})
						Expect(err).To(MatchError("getting source id: some-get-error"))
					})
				})
			})

			Context("when getting the destination group id fails", func() {
				Context("when the error is because the destination group does not exist", func() {
					BeforeEach(func() {
						fakeGroup.GetIDStub = func(store.Transaction, string) (int, error) {
							if fakeGroup.GetIDCallCount() == 2 {
								return -1, sql.ErrNoRows
							}
							return 0, nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{
							models.Policy{Source: models.Source{ID: "peach"}, Destination: models.Destination{ID: "pear"}},
							models.Policy{Source: models.Source{ID: "apple"}, Destination: models.Destination{ID: "banana"}},
						})
						Expect(err).NotTo(HaveOccurred())
						Expect(fakeGroup.GetIDCallCount()).To(Equal(4))

						_, call0SourceID := fakeGroup.GetIDArgsForCall(0)
						Expect(call0SourceID).To(Equal("peach"))

						_, call1SourceID := fakeGroup.GetIDArgsForCall(1)
						Expect(call1SourceID).To(Equal("pear"))

						_, call2SourceID := fakeGroup.GetIDArgsForCall(2)
						Expect(call2SourceID).To(Equal("apple"))

						_, call3SourceID := fakeGroup.GetIDArgsForCall(3)
						Expect(call3SourceID).To(Equal("banana"))

						Expect(fakeDestination.GetIDCallCount()).To(Equal(1))
					})
				})

				Context("when the error is for any other reason", func() {
					BeforeEach(func() {
						fakeGroup.GetIDStub = func(store.Transaction, string) (int, error) {
							if fakeGroup.GetIDCallCount() > 1 {
								return -1, errors.New("some-get-error")
							}
							return 0, nil
						}
					})
					It("returns a error", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{{
							Source: models.Source{ID: "some-app-guid"},
							Destination: models.Destination{
								ID:       "some-other-app-guid",
								Protocol: "tcp",
								Port:     8080,
							},
						}})
						Expect(err).To(MatchError("getting destination group id: some-get-error"))
					})
				})
			})

			Context("when getting the destination id fails", func() {
				Context("when the error is because the destination does not exist", func() {
					BeforeEach(func() {
						fakeDestination.GetIDStub = func(store.Transaction, int, int, string) (int, error) {
							if fakeDestination.GetIDCallCount() == 1 {
								return -1, sql.ErrNoRows
							}
							return 0, nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{
							models.Policy{Source: models.Source{ID: "peach"}, Destination: models.Destination{ID: "pear"}},
							models.Policy{Source: models.Source{ID: "apple"}, Destination: models.Destination{ID: "banana"}},
						})
						Expect(err).NotTo(HaveOccurred())
						Expect(fakePolicy.DeleteCallCount()).To(Equal(1))
					})
				})

				Context("when the error is for any other reason", func() {
					BeforeEach(func() {
						fakeDestination.GetIDReturns(-1, errors.New("some-dest-id-get-error"))
					})

					It("returns a error", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{{
							Source: models.Source{ID: "some-app-guid"},
							Destination: models.Destination{
								ID:       "some-other-app-guid",
								Protocol: "tcp",
								Port:     8080,
							},
						}})
						Expect(err).To(MatchError("getting destination id: some-dest-id-get-error"))
					})
				})
			})

			Context("when deleting the policy fails", func() {
				Context("when the error is because the policy does not exist", func() {
					BeforeEach(func() {
						fakePolicy.DeleteStub = func(store.Transaction, int, int) error {
							if fakePolicy.DeleteCallCount() == 1 {
								return sql.ErrNoRows
							}
							return nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{
							models.Policy{Source: models.Source{ID: "peach"}, Destination: models.Destination{ID: "pear"}},
							models.Policy{Source: models.Source{ID: "apple"}, Destination: models.Destination{ID: "banana"}},
						})
						Expect(err).NotTo(HaveOccurred())
						Expect(fakePolicy.DeleteCallCount()).To(Equal(2))
					})
				})

				Context("when the error is for any other reason", func() {
					BeforeEach(func() {
						fakePolicy.DeleteReturns(errors.New("some-delete-error"))
					})

					It("returns a error", func() {
						err = dataStore.Delete(context.Background(), []models.Policy{{
							Source: models.Source{ID: "some-app-guid"},
							Destination: models.Destination{
								ID:       "some-other-app-guid",
								Protocol: "tcp",
								Port:     8080,
							},
						}})
						Expect(err).To(MatchError("deleting policy: some-delete-error"))
					})
				})
			})

			Context("when counting policies by destination_id fails", func() {
				BeforeEach(func() {
					fakePolicy.CountWhereDestinationIDReturns(0, errors.New("some-dst-count-error"))
				})

				It("returns a error", func() {
					err = dataStore.Delete(context.Background(), []models.Policy{{
						Source: models.Source{ID: "some-app-guid"},
						Destination: models.Destination{
							ID:       "some-other-app-guid",
							Protocol: "tcp",
							Port:     8080,
						},
					}})
					Expect(err).To(MatchError("counting destination id: some-dst-count-error"))
				})
			})

			Context("when deleting a destination fails", func() {
				BeforeEach(func() {
					fakeDestination.DeleteReturns(errors.New("some-dst-delete-error"))
				})

				It("returns a error", func() {
					err = dataStore.Delete(context.Background(), []models.Policy{{
						Source: models.Source{ID: "some-app-guid"},
						Destination: models.Destination{
							ID:       "some-other-app-guid",
							Protocol: "tcp",
							Port:     8080,
						},
					}})
					Expect(err).To(MatchError("deleting destination: some-dst-delete-error"))
				})
			})

			Context("when counting policies by group_id fails", func() {
				BeforeEach(func() {
					fakePolicy.CountWhereGroupIDReturns(-1, errors.New("some-group-id-count-error"))
				})

				It("returns a error", func() {
					err = dataStore.Delete(context.Background(), []models.Policy{{
						Source: models.Source{ID: "some-app-guid"},
						Destination: models.Destination{
							ID:       "some-other-app-guid",
							Protocol: "tcp",
							Port:     8080,
						},
					}})
					Expect(err).To(MatchError("deleting group row: some-group-id-count-error"))
				})
			})

			Context("when counting destinations by group_id fails", func() {
				BeforeEach(func() {
					fakeDestination.CountWhereGroupIDReturns(-1, errors.New("some-dst-count-error"))
				})

				It("returns a error", func() {
					err = dataStore.Delete(context.Background(), []models.Policy{{
						Source: models.Source{ID: "some-app-guid"},
						Destination: models.Destination{
							ID:       "some-other-app-guid",
							Protocol: "tcp",
							Port:     8080,
						},
					}})
					Expect(err).To(MatchError("deleting group row: some-dst-count-error"))
				})
			})

			Context("when deleting the group fails", func() {
				BeforeEach(func() {
					fakeGroup.DeleteReturns(errors.New("some-group-delete-error"))
				})

				It("returns a error", func() {
					err = dataStore.Delete(context.Background(), []models.Policy{{
						Source: models.Source{ID: "some-app-guid"},
						Destination: models.Destination{
							ID:       "some-other-app-guid",
							Protocol: "tcp",
							Port:     8080,
						},
					}})
					Expect(err).To(MatchError("deleting group row: some-group-delete-error"))
				})
			})

			Context("when the context gets cancelled", func() {
				It("returns a error", func() {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
					cancel()
					err := dataStore.Delete(ctx, []models.Policy{{
						Source: models.Source{ID: "some-app-guid"},
						Destination: models.Destination{
							ID:       "some-other-app-guid",
							Protocol: "tcp",
							Port:     8080,
						},
					}})
					Expect(err).To(MatchError("context done"))
				})
			})
		})
	})
})
