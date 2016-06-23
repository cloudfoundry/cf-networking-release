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

		dataStore, err = store.New(realDb, group, destination, policy)
		Expect(err).NotTo(HaveOccurred())
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

	Describe("Connecting to the database and migrating", func() {
		Context("when the tables already exist", func() {
			It("succeeds", func() {
				_, err := store.New(realDb, group, destination, policy)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.ExecReturns(nil, errors.New("some error"))
			})

			It("should return a sensible error", func() {
				_, err := store.New(mockDb, group, destination, policy)
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

		Context("when a transaction create fails", func() {
			var err error

			BeforeEach(func() {
				mockDb.BeginxReturns(nil, errors.New("some-db-error"))
				dataStore, err = store.New(mockDb, group, destination, policy)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err = dataStore.Create(nil)
				Expect(err).To(MatchError("begin transaction: some-db-error"))
			})
		})

		Context("when a Group create record fails", func() {
			var fakeGroup *fakes.GroupRepo
			var err error

			BeforeEach(func() {
				fakeGroup = &fakes.GroupRepo{}
				fakeGroup.CreateReturns(-1, errors.New("some-insert-error"))

				dataStore, err = store.New(realDb, fakeGroup, destination, policy)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a error", func() {
				err = dataStore.Create([]models.Policy{{
					Source: models.Source{"some-app-guid"},
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

				dataStore, err = store.New(realDb, fakeGroup, destination, policy)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				err = dataStore.Create([]models.Policy{{
					Source: models.Source{"some-app-guid"},
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

				dataStore, err = store.New(realDb, group, fakeDestination, policy)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a error", func() {
				err = dataStore.Create([]models.Policy{{
					Source: models.Source{"some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})
				Expect(err).To(MatchError("creating destination: some-insert-error"))
				var groupsCount int
				err = realDb.QueryRow(`select count(*) from groups`).Scan(&groupsCount)
				Expect(groupsCount).To(BeZero())
			})
		})

		Context("when a Policy create record fails", func() {
			var fakePolicy *fakes.PolicyRepo
			var err error

			BeforeEach(func() {
				fakePolicy = &fakes.PolicyRepo{}
				fakePolicy.CreateReturns(errors.New("some-insert-error"))

				dataStore, err = store.New(realDb, group, destination, fakePolicy)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a error", func() {
				err = dataStore.Create([]models.Policy{{
					Source: models.Source{"some-app-guid"},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Protocol: "tcp",
						Port:     8080,
					},
				}})
				Expect(err).To(MatchError("creating policy: some-insert-error"))
			})
		})
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
				store, err := store.New(mockDb, group, destination, policy)
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

				_, err = store.New(realDb, group, destination, policy)
				Expect(err).NotTo(HaveOccurred())
				rows, err = realDb.Query(`select * from policies`)
				Expect(err).NotTo(HaveOccurred())

				mockDb.QueryReturns(rows, nil)
			})

			AfterEach(func() {
				rows.Close()
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb, group, destination, policy)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.All()
				Expect(err).To(MatchError(ContainSubstring("listing all: sql: expected")))
			})
		})
	})

	Describe("Tags", func() {
		BeforeEach(func() {
			policies := []models.Policy{{
				Source: models.Source{"some-app-guid"},
				Destination: models.Destination{
					ID:       "some-other-app-guid",
					Protocol: "tcp",
					Port:     8080,
				},
			}, {
				Source: models.Source{"some-app-guid"},
				Destination: models.Destination{
					ID:       "another-app-guid",
					Protocol: "udp",
					Port:     5555,
				},
			}}

			err := dataStore.Create(policies)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all tags that have been added", func() {
			tags, err := dataStore.Tags()
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(ConsistOf([]models.Tag{
				{ID: "some-app-guid", Tag: "0001"},
				{ID: "some-other-app-guid", Tag: "0002"},
				{ID: "another-app-guid", Tag: "0003"},
			}))
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.QueryReturns(nil, errors.New("some query error"))
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb, group, destination, policy)
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
				store, err := store.New(mockDb, group, destination, policy)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Tags()
				Expect(err).To(MatchError(ContainSubstring("listing tags: sql: expected")))
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			err := dataStore.Create([]models.Policy{
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
			err := dataStore.Delete([]models.Policy{{
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
				Source: models.Source{ID: "another-app-guid"},
				Destination: models.Destination{
					ID:       "yet-another-app-guid",
					Protocol: "udp",
					Port:     5555,
				},
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
				dataStore, err = store.New(realDb, fakeGroup, fakeDestination, fakePolicy)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when a transaction create fails", func() {
				var err error

				BeforeEach(func() {
					mockDb.BeginxReturns(nil, errors.New("some-db-error"))
					dataStore, err = store.New(mockDb, group, destination, policy)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					err = dataStore.Delete(nil)
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
						err = dataStore.Delete([]models.Policy{
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
						err = dataStore.Delete([]models.Policy{{
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
						err = dataStore.Delete([]models.Policy{
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
						err = dataStore.Delete([]models.Policy{{
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
						err = dataStore.Delete([]models.Policy{
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
						err = dataStore.Delete([]models.Policy{{
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
						err = dataStore.Delete([]models.Policy{
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
						err = dataStore.Delete([]models.Policy{{
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
		})
	})
})
