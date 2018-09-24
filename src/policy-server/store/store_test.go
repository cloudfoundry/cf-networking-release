package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	"policy-server/store"
	"policy-server/store/fakes"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"

	"code.cloudfoundry.org/cf-networking-helpers/testsupport"

	"policy-server/store/migrations"

	"test-helpers"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var (
		dataStore    store.Store
		tagDataStore store.TagStore
		dbConf       db.Config
		realDb       *db.ConnWrapper
		mockDb       *fakes.Db
		group        store.GroupRepo
		destination  store.DestinationRepo
		policy       store.PolicyRepo
		tx           *dbfakes.Transaction

		tagLength int
	)
	const NumAttempts = 5

	BeforeEach(func() {
		mockDb = &fakes.Db{}

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("store_test_node_%d", time.Now().UnixNano())

		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Store Test")

		var err error
		realDb, err = db.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Store Test", "Store Test", logger)
		Expect(err).NotTo(HaveOccurred())

		group = &store.GroupTable{}
		destination = &store.DestinationTable{}
		policy = &store.PolicyTable{}
		tx = &dbfakes.Transaction{}

		mockDb.DriverNameReturns(realDb.DriverName())
		mockDb.BeginxReturns(tx, nil)
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testhelpers.RemoveDatabase(dbConf)
	})

	Describe("concurrent create and delete requests", func() {
		retry := func(dataStore store.Store, crud string, p store.Policy) error {
			var err error
			for attempt := 0; attempt < NumAttempts; attempt++ {
				time.Sleep(time.Duration(attempt) * time.Second)
				switch crud {
				case "create":
					err = dataStore.Create([]store.Policy{p})
				case "delete":
					err = dataStore.Delete([]store.Policy{p})
				}
				if err == nil {
					break
				} else {
					fmt.Printf("Error on %s attempt. Retrying %d of %d: %s", crud, attempt, NumAttempts, err)
				}
			}
			return err
		}
		It("remains consistent", func() {
			migrateAndPopulateTags(realDb, 2)
			dataStore := store.New(realDb, group, destination, policy, 2)

			nPolicies := 1000
			var policies []interface{}
			for i := 0; i < nPolicies; i++ {
				appName := fmt.Sprintf("some-app-%x", i)
				policies = append(policies, store.Policy{
					Source:      store.Source{ID: appName},
					Destination: store.Destination{ID: appName, Protocol: "tcp", Port: 1234},
				})
			}

			parallelRunner := &testsupport.ParallelRunner{
				NumWorkers: 4,
			}
			toDelete := make(chan interface{}, nPolicies)

			go func() {
				parallelRunner.RunOnSlice(policies, func(policy interface{}) {
					p := policy.(store.Policy)
					Expect(retry(dataStore, "create", p)).To(Succeed())
					toDelete <- p
				})
				close(toDelete)
			}()

			var nDeleted int32
			parallelRunner.RunOnChannel(toDelete, func(policy interface{}) {
				p := policy.(store.Policy)
				Expect(retry(dataStore, "delete", p)).To(Succeed())
				atomic.AddInt32(&nDeleted, 1)
			})

			Expect(nDeleted).To(Equal(int32(nPolicies)))

			allPolicies, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())

			Expect(allPolicies).To(BeEmpty())
		})
	})

	Describe("Create", func() {
		BeforeEach(func() {
			tagLength = 1
			migrateAndPopulateTags(realDb, tagLength)
			dataStore = store.New(realDb, group, destination, policy, tagLength)
			tagDataStore = store.NewTagStore(realDb, group, tagLength)
		})

		It("saves the policies", func() {
			policies := []store.Policy{{
				Source: store.Source{ID: "some-app-guid"},
				Destination: store.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Ports: store.Ports{
						Start: 8080,
						End:   9000,
					},
				},
			}, {
				Source: store.Source{ID: "another-app-guid"},
				Destination: store.Destination{
					ID:       "some-other-app-guid",
					Protocol: "udp",
					Ports: store.Ports{
						Start: 123,
						End:   123,
					},
				},
			}}

			err := dataStore.Create(policies)
			Expect(err).NotTo(HaveOccurred())

			p, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(p)).To(Equal(2))
		})

		Context("when a transaction begin fails", func() {
			var err error

			BeforeEach(func() {
				mockDb.BeginxReturns(nil, errors.New("some-db-error"))
				dataStore = store.New(mockDb, group, destination, policy, 2)
			})

			It("returns an error", func() {
				err = dataStore.Create(nil)
				Expect(err).To(MatchError("create transaction: some-db-error"))
			})
		})

		Context("when the createWithTx fails", func() {
			It("rollsback the transaction", func() {
				fakeGroup := &fakes.GroupRepo{}
				fakeGroup.CreateReturns(-1, errors.New("failed to create group"))

				dataStore := store.New(mockDb, fakeGroup, destination, policy, 2)

				err := dataStore.Create([]store.Policy{{}})
				Expect(err).To(MatchError("creating group: failed to create group"))
				Expect(tx.RollbackCallCount()).To(Equal(1))
			})
		})

		Context("when commiting the transacton fails", func() {
			It("returns the error", func() {

				fakeGroup := &fakes.GroupRepo{}
				fakeDestination := &fakes.DestinationRepo{}
				fakePolicy := &fakes.PolicyRepo{}

				tx.CommitReturns(errors.New("commit failure"))

				dataStore := store.New(mockDb, fakeGroup, fakeDestination, fakePolicy, 2)
				err := dataStore.Create([]store.Policy{{}})
				Expect(err).To(MatchError("commit transaction: commit failure"))
			})
		})

		Context("when a policy with the same content already exists", func() {
			It("does not duplicate table rows", func() {
				policies := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Ports: store.Ports{
							Start: 7000,
							End:   8000,
						},
					},
				}}

				err := dataStore.Create(policies)
				Expect(err).NotTo(HaveOccurred())

				p, err := dataStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(p)).To(Equal(1))

				policyDuplicate := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Ports: store.Ports{
							Start: 7000,
							End:   8000,
						},
					},
				}}

				err = dataStore.Create(policyDuplicate)
				Expect(err).NotTo(HaveOccurred())

				p, err = dataStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(p)).To(Equal(1))
			})
		})

		Context("when there are no tags left to allocate", func() {
			BeforeEach(func() {
				var policies []store.Policy
				for i := 1; i < 256; i++ {
					policies = append(policies, store.Policy{
						Source: store.Source{ID: fmt.Sprintf("%d", i)},
						Destination: store.Destination{
							ID:       fmt.Sprintf("%d", i),
							Protocol: "tcp",
							Port:     8080,
						},
					})
				}
				err := dataStore.Create(policies)
				Expect(err).NotTo(HaveOccurred())

				Expect(dataStore.All()).To(HaveLen(255))
			})
			It("returns an error", func() {
				policies := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err := dataStore.Create(policies)
				Expect(err).To(MatchError(ContainSubstring("failed to find available tag")))
			})
		})

		Context("when a tag is freed by delete", func() {
			It("reuses the tag", func() {
				policies := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}, {
					Source: store.Source{ID: "another-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "udp",
						Port:     1234,
					},
				}}

				err := dataStore.Create(policies)
				Expect(err).NotTo(HaveOccurred())

				tags, err := tagDataStore.Tags()
				Expect(err).NotTo(HaveOccurred())
				Expect(tags).To(ConsistOf([]store.Tag{
					{ID: "some-app-guid", Tag: "01", Type: "app"},
					{ID: "some-other-app-guid", Tag: "02", Type: "app"},
					{ID: "another-app-guid", Tag: "03", Type: "app"},
				}))

				err = dataStore.Delete(policies[:1])
				Expect(err).NotTo(HaveOccurred())

				newPolicies := []store.Policy{{
					Source: store.Source{ID: "yet-another-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err = dataStore.Create(newPolicies)
				Expect(err).NotTo(HaveOccurred())

				Expect(err).NotTo(HaveOccurred())

				tags, err = tagDataStore.Tags()
				Expect(err).NotTo(HaveOccurred())
				Expect(tags).To(ConsistOf([]store.Tag{
					{ID: "yet-another-app-guid", Tag: "01", Type: "app"},
					{ID: "some-other-app-guid", Tag: "02", Type: "app"},
					{ID: "another-app-guid", Tag: "03", Type: "app"},
				}))
			})
		})

		Context("when a Group create record fails", func() {
			var fakeGroup *fakes.GroupRepo
			var err error

			BeforeEach(func() {
				fakeGroup = &fakes.GroupRepo{}
				fakeGroup.CreateReturns(-1, errors.New("some-insert-error"))
				migrateAndPopulateTags(realDb, 2)

				dataStore = store.New(realDb, fakeGroup, destination, policy, 2)
			})

			It("returns a error", func() {
				policies := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err = dataStore.Create(policies)

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
				fakeGroup.CreateStub = func(t db.Transaction, guid, groupType string) (int, error) {
					response := responses[0]
					responses = responses[1:]
					return response.Id, response.Err
				}

				migrateAndPopulateTags(realDb, 2)
				dataStore = store.New(realDb, fakeGroup, destination, policy, 2)
			})

			It("returns the error", func() {
				policies := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err = dataStore.Create(policies)

				Expect(err).To(MatchError("creating group: some-insert-error"))
			})
		})

		Context("when a Destination create record fails", func() {
			var fakeDestination *fakes.DestinationRepo
			var err error

			BeforeEach(func() {
				fakeDestination = &fakes.DestinationRepo{}
				fakeDestination.CreateReturns(-1, errors.New("some-insert-error"))

				migrateAndPopulateTags(realDb, 2)
				dataStore = store.New(realDb, group, fakeDestination, policy, 2)
			})

			It("returns a error", func() {
				policies := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err = dataStore.Create(policies)

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

				migrateAndPopulateTags(realDb, 2)
				dataStore = store.New(realDb, group, destination, fakePolicy, 2)
			})

			It("returns a error", func() {
				policies := []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err = dataStore.Create(policies)

				Expect(err).To(MatchError("creating policy: some-insert-error"))
			})
		})
	})

	Describe("All", func() {
		var expectedPolicies []store.Policy

		BeforeEach(func() {
			var err error
			expectedPolicies = []store.Policy{{
				Source: store.Source{ID: "some-app-guid", Tag: "01"},
				Destination: store.Destination{
					ID:       "some-other-app-guid",
					Tag:      "02",
					Protocol: "tcp",
					Ports: store.Ports{
						Start: 5000,
						End:   6000,
					},
				},
			}}
			migrateAndPopulateTags(realDb, 1)
			dataStore = store.New(realDb, group, destination, policy, 1)

			err = dataStore.Create(expectedPolicies)
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
				store := store.New(mockDb, group, destination, policy, 2)

				_, err := store.All()
				Expect(err).To(MatchError("listing all: some query error"))
			})
		})

		Context("when the query result parsing fails", func() {
			var rows *sql.Rows

			BeforeEach(func() {
				expectedPolicies = []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err := dataStore.Create(expectedPolicies)
				Expect(err).NotTo(HaveOccurred())

				store.New(realDb, group, destination, policy, 2)

				rows, err = realDb.Query(`select * from policies`)
				Expect(err).NotTo(HaveOccurred())

				mockDb.QueryReturns(rows, nil)
			})

			AfterEach(func() {
				rows.Close()
			})

			It("should return a sensible error", func() {
				store := store.New(mockDb, group, destination, policy, 2)
				_, err := store.All()
				Expect(err).To(MatchError(ContainSubstring("listing all: sql: expected")))
			})
		})
	})

	Describe("ByGuids", func() {
		var allPolicies []store.Policy
		var expectedPolicies []store.Policy
		var err error

		BeforeEach(func() {
			allPolicies = []store.Policy{
				{
					Source: store.Source{
						ID:  "app-guid-00",
						Tag: "01",
					},
					Destination: store.Destination{
						ID:       "app-guid-01",
						Tag:      "02",
						Protocol: "tcp",
						Port:     101,
						Ports: store.Ports{
							Start: 101,
							End:   101,
						},
					},
				},
				{
					Source: store.Source{
						ID:  "app-guid-01",
						Tag: "02",
					},
					Destination: store.Destination{
						ID:       "app-guid-02",
						Tag:      "03",
						Protocol: "tcp",
						Port:     102,
						Ports: store.Ports{
							Start: 102,
							End:   102,
						},
					},
				},
				{
					Source: store.Source{
						ID:  "app-guid-02",
						Tag: "03",
					},
					Destination: store.Destination{
						ID:       "app-guid-00",
						Tag:      "01",
						Protocol: "tcp",
						Port:     103,
						Ports: store.Ports{
							Start: 103,
							End:   103,
						},
					},
				},
				{
					Source: store.Source{
						ID:  "app-guid-03",
						Tag: "04",
					},
					Destination: store.Destination{
						ID:       "app-guid-03",
						Tag:      "04",
						Protocol: "tcp",
						Port:     104,
						Ports: store.Ports{
							Start: 104,
							End:   104,
						},
					},
				},
			}

			migrateAndPopulateTags(realDb, 1)

			dataStore = store.New(realDb, group, destination, policy, 1)

			err := dataStore.Create(allPolicies)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when empty args is provided", func() {
			BeforeEach(func() {
				dataStore = store.New(mockDb, group, destination, policy, 1)
			})

			It("returns an empty slice ", func() {
				policies, err := dataStore.ByGuids(nil, nil, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(BeEmpty())

				By("not making any queries")
				Expect(mockDb.QueryCallCount()).To(Equal(0))
			})
		})

		Context("when srcGuids is provided", func() {
			It("returns policies whose source is in srcGuids", func() {
				policies, err := dataStore.ByGuids([]string{"app-guid-00", "app-guid-01"}, nil, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(allPolicies[0], allPolicies[1]))
			})
		})

		Context("when destGuids is provided", func() {
			It("returns policies whose destination is in destGuids", func() {
				policies, err := dataStore.ByGuids(nil, []string{"app-guid-00", "app-guid-01"}, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(allPolicies[0], allPolicies[2]))
			})
		})

		Context("when srcGuids and destGuids are provided", func() {
			It("returns policies that satisfy either srcGuids or destGuids", func() {
				policies, err := dataStore.ByGuids(
					[]string{"app-guid-00", "app-guid-01"},
					[]string{"app-guid-00", "app-guid-01"},
					false,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(
					allPolicies[0], allPolicies[1], allPolicies[2],
				))
			})
		})

		Context("when inSourceAndDest is true", func() {
			It("returns policies that are in srcGuids and destGuids", func() {
				policies, err := dataStore.ByGuids(
					[]string{"app-guid-00", "app-guid-01"},
					[]string{"app-guid-00", "app-guid-01"},
					true,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(
					allPolicies[0],
				))
			})
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.QueryReturns(nil, errors.New("some query error"))
			})

			It("should return a sensible error", func() {
				store := store.New(mockDb, group, destination, policy, 2)

				_, err = store.ByGuids(
					[]string{"does-not-matter"},
					[]string{"does-not-matter"},
					false,
				)
				Expect(err).To(MatchError("listing all: some query error"))
			})
		})

		Context("when the query result parsing fails", func() {
			var rows *sql.Rows

			BeforeEach(func() {
				expectedPolicies = []store.Policy{{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}}

				err := dataStore.Create(expectedPolicies)
				Expect(err).NotTo(HaveOccurred())

				store.New(realDb, group, destination, policy, 2)
				rows, err = realDb.Query(`select * from policies`)
				Expect(err).NotTo(HaveOccurred())

				mockDb.QueryReturns(rows, nil)
			})

			AfterEach(func() {
				rows.Close()
			})

			It("should return a sensible error", func() {
				store := store.New(mockDb, group, destination, policy, 2)

				_, err = store.ByGuids(
					[]string{"does-not-matter"},
					[]string{"does-not-matter"},
					true,
				)
				Expect(err).To(MatchError(ContainSubstring("listing all: sql: expected")))
			})
		})
	})

	Describe("CheckDatabase", func() {
		BeforeEach(func() {
			migrateAndPopulateTags(realDb, 1)
			dataStore = store.New(realDb, group, destination, policy, 1)
		})

		It("checks that the database exists", func() {
			err := dataStore.CheckDatabase()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the database connection is closed", func() {
			BeforeEach(func() {
				if realDb != nil {
					Expect(realDb.Close()).To(Succeed())
				}
			})

			It("returns an error", func() {
				err := dataStore.CheckDatabase()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("sql: database is closed"))
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			tagLength = 1
			migrateAndPopulateTags(realDb, tagLength)
			dataStore = store.New(realDb, group, destination, policy, tagLength)
			tagDataStore = store.NewTagStore(realDb, group, tagLength)

			policies := []store.Policy{
				{
					Source: store.Source{ID: "some-app-guid"},
					Destination: store.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				},
				{
					Source: store.Source{ID: "another-app-guid"},
					Destination: store.Destination{
						ID:       "yet-another-app-guid",
						Protocol: "udp",
						Port:     5555,
					},
				},
			}

			err := dataStore.Create(policies)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes the specified policies", func() {
			err := dataStore.Delete([]store.Policy{{
				Source: store.Source{ID: "some-app-guid"},
				Destination: store.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			policies, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(Equal([]store.Policy{{
				Source: store.Source{ID: "another-app-guid", Tag: "03"},
				Destination: store.Destination{
					ID:       "yet-another-app-guid",
					Protocol: "udp",
					Port:     5555,
					Tag:      "04",
				},
			}}))
		})

		It("deletes the tags if no longer referenced", func() {
			err := dataStore.Delete([]store.Policy{{
				Source: store.Source{ID: "some-app-guid"},
				Destination: store.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			policies, err := tagDataStore.Tags()
			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(Equal([]store.Tag{{
				ID:   "another-app-guid",
				Tag:  "03",
				Type: "app",
			}, {
				ID:   "yet-another-app-guid",
				Tag:  "04",
				Type: "app",
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
				migrateAndPopulateTags(realDb, 2)
				dataStore = store.New(realDb, fakeGroup, fakeDestination, fakePolicy, 2)
			})

			Context("when a transaction begin fails", func() {
				var err error

				BeforeEach(func() {
					mockDb.BeginxReturns(nil, errors.New("some-db-error"))
					dataStore = store.New(mockDb, group, destination, policy, 2)
				})

				It("returns an error", func() {
					err = dataStore.Delete(nil)
					Expect(err).To(MatchError("create transaction: some-db-error"))
				})
			})

			Context("when commiting fails", func() {
				It("returns the error", func() {
					tx.CommitReturns(errors.New("failed to commit"))
					dataStore := store.New(mockDb, fakeGroup, fakeDestination, fakePolicy, 2)
					err := dataStore.Delete([]store.Policy{{}})
					Expect(err).To(MatchError("commit transaction: failed to commit"))
				})
			})

			Context("when the deleteWithTx fails", func() {
				It("rollsback the transaction", func() {
					fakeGroup.GetIDReturns(-1, errors.New("failed to get id"))
					dataStore := store.New(mockDb, fakeGroup, fakeDestination, fakePolicy, 2)

					err := dataStore.Delete([]store.Policy{{}})
					Expect(err).To(MatchError("getting source id: failed to get id"))
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("when getting the source id fails", func() {
				Context("when the error is because the source does not exist", func() {
					BeforeEach(func() {
						fakeGroup.GetIDStub = func(db.Transaction, string) (int, error) {
							if fakeGroup.GetIDCallCount() == 1 {
								return -1, sql.ErrNoRows
							}
							return 0, nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete([]store.Policy{
							{Source: store.Source{ID: "0"}},
							{Source: store.Source{ID: "apple"}, Destination: store.Destination{ID: "banana"}},
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
						err = dataStore.Delete([]store.Policy{{
							Source: store.Source{ID: "some-app-guid"},
							Destination: store.Destination{
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
						fakeGroup.GetIDStub = func(db.Transaction, string) (int, error) {
							if fakeGroup.GetIDCallCount() == 2 {
								return -1, sql.ErrNoRows
							}
							return 0, nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete([]store.Policy{
							{Source: store.Source{ID: "peach"}, Destination: store.Destination{ID: "pear"}},
							{Source: store.Source{ID: "apple"}, Destination: store.Destination{ID: "banana"}},
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
						fakeGroup.GetIDStub = func(db.Transaction, string) (int, error) {
							if fakeGroup.GetIDCallCount() > 1 {
								return -1, errors.New("some-get-error")
							}
							return 0, nil
						}
					})
					It("returns a error", func() {
						err = dataStore.Delete([]store.Policy{{
							Source: store.Source{ID: "some-app-guid"},
							Destination: store.Destination{
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
						fakeDestination.GetIDStub = func(db.Transaction, int, int, int, int, string) (int, error) {
							if fakeDestination.GetIDCallCount() == 1 {
								return -1, sql.ErrNoRows
							}
							return 0, nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete([]store.Policy{
							{Source: store.Source{ID: "peach"}, Destination: store.Destination{ID: "pear"}},
							{Source: store.Source{ID: "apple"}, Destination: store.Destination{ID: "banana"}},
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
						err = dataStore.Delete([]store.Policy{{
							Source: store.Source{ID: "some-app-guid"},
							Destination: store.Destination{
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
						fakePolicy.DeleteStub = func(db.Transaction, int, int) error {
							if fakePolicy.DeleteCallCount() == 1 {
								return sql.ErrNoRows
							}
							return nil
						}
					})

					It("swallows the error and continues", func() {
						err = dataStore.Delete([]store.Policy{
							{Source: store.Source{ID: "peach"}, Destination: store.Destination{ID: "pear"}},
							{Source: store.Source{ID: "apple"}, Destination: store.Destination{ID: "banana"}},
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
						err = dataStore.Delete([]store.Policy{{
							Source: store.Source{ID: "some-app-guid"},
							Destination: store.Destination{
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
					err = dataStore.Delete([]store.Policy{{
						Source: store.Source{ID: "some-app-guid"},
						Destination: store.Destination{
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
					err = dataStore.Delete([]store.Policy{{
						Source: store.Source{ID: "some-app-guid"},
						Destination: store.Destination{
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
					err = dataStore.Delete([]store.Policy{{
						Source: store.Source{ID: "some-app-guid"},
						Destination: store.Destination{
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
					err = dataStore.Delete([]store.Policy{{
						Source: store.Source{ID: "some-app-guid"},
						Destination: store.Destination{
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
					err = dataStore.Delete([]store.Policy{{
						Source: store.Source{ID: "some-app-guid"},
						Destination: store.Destination{
							ID:       "some-other-app-guid",
							Protocol: "tcp",
							Port:     8080,
						},
					}})
					Expect(err).To(MatchError("deleting group row: some-group-delete-error"))
				})
			})
		})
	})
})

func migrateAndPopulateTags(realDb *db.ConnWrapper, tl int) {
	migrate(realDb)

	tagPopulator := &store.TagPopulator{DBConnection: realDb}
	err := tagPopulator.PopulateTables(tl)
	Expect(err).NotTo(HaveOccurred())
}

func migrate(realDb *db.ConnWrapper) {
	migrator := &migrations.Migrator{
		MigrateAdapter: &migrations.MigrateAdapter{},
		MigrationsProvider: &migrations.MigrationsProvider{
			Store: &store.MigrationsStore{
				DBConn: realDb,
			},
		},
	}
	_, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 0)
	Expect(err).ToNot(HaveOccurred())
}
