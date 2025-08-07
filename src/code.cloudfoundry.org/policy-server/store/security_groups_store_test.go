package store_test

import (
	"errors"
	"fmt"
	"time"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/store/fakes"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecurityGroupsStore", func() {
	var (
		securityGroupsStore *store.SGStore
		dbConf              dbHelper.Config
		realDb              *dbHelper.ConnWrapper
	)

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("security_groups_store_test_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lagertest.NewTestLogger("test")

		var err error
		realDb, err = dbHelper.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Security Groups Store Test", "Security Groups Store Test", logger)
		Expect(err).NotTo(HaveOccurred())
		securityGroupsStore = &store.SGStore{
			Conn:   realDb,
			Logger: logger,
		}

		migrate(realDb)
	})

	AfterEach(func() {
		Expect(realDb.Close()).To(Succeed())
		testhelpers.RemoveDatabase(dbConf)
	})

	Describe("BySpaceGuids", func() {
		var securityGroups []store.SecurityGroup

		BeforeEach(func() {
			securityGroups = []store.SecurityGroup{{
				Guid:              "first-guid",
				Name:              "first-asg",
				Rules:             "firstRules",
				RunningSpaceGuids: []string{"space-a"},
			}, {
				Guid:              "second-guid",
				Name:              "second-name",
				Rules:             "secondRules",
				RunningSpaceGuids: []string{"space-b"},
				StagingSpaceGuids: []string{"space-b"},
			}, {
				Guid:              "third-guid",
				Name:              "third-name",
				Rules:             "thirdRules",
				RunningSpaceGuids: []string{"space-c", "space-d", "space-e"},
				StagingSpaceGuids: []string{"space-c", "space-d", "space-f"},
			}, {
				Guid:              "fourth-guid",
				Name:              "fourth-name",
				Rules:             "fourthRules",
				RunningSpaceGuids: []string{"space-d"},
				StagingSpaceGuids: []string{"space-d"},
			}}

			err := securityGroupsStore.Replace(securityGroups)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when no space guids are provided", func() {
			It("returns empty list", func() {
				securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(0))
			})
		})

		Context("search by staging space guid", func() {
			It("fetches global asgs and asgs attached to provided spaces", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-b"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(1))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "second-guid",
					Name:              "second-name",
					Rules:             "secondRules",
					RunningSpaceGuids: []string{"space-b"},
					StagingSpaceGuids: []string{"space-b"},
				}))
				Expect(pagination.Next).To(Equal(0))
			})
		})

		Context("search by running space guid", func() {
			It("fetches global asgs and asgs attached to provided spaces", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-a"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(1))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					RunningSpaceGuids: []string{"space-a"},
				}))
				Expect(pagination.Next).To(Equal(0))
			})
		})

		Context("when one of the spaces of the security group wth multiple spaces is requested", func() {
			It("returns that security group", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-e"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(securityGroups)).To(Equal(1))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "third-guid",
					Name:              "third-name",
					Rules:             "thirdRules",
					RunningSpaceGuids: []string{"space-c", "space-d", "space-e"},
					StagingSpaceGuids: []string{"space-c", "space-d", "space-f"},
				}))
				Expect(pagination.Next).To(Equal(0))
			})
		})

		Context("when the space that has multiple groups is requested", func() {
			It("returns all security groups in that space, ordered by id", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-d"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(securityGroups)).To(Equal(2))
				Expect(securityGroups).To(Equal([]store.SecurityGroup{
					{
						Guid:              "third-guid",
						Name:              "third-name",
						Rules:             "thirdRules",
						RunningSpaceGuids: []string{"space-c", "space-d", "space-e"},
						StagingSpaceGuids: []string{"space-c", "space-d", "space-f"},
					}, {
						Guid:              "fourth-guid",
						Name:              "fourth-name",
						Rules:             "fourthRules",
						RunningSpaceGuids: []string{"space-d"},
						StagingSpaceGuids: []string{"space-d"},
					}}))
				Expect(pagination.Next).To(Equal(0))
			})
		})

		Context("when multiple spaces are requested", func() {
			It("returns all security groups in all requested spaces", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-e", "space-d"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(securityGroups)).To(Equal(2))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "third-guid",
					Name:              "third-name",
					Rules:             "thirdRules",
					RunningSpaceGuids: []string{"space-c", "space-d", "space-e"},
					StagingSpaceGuids: []string{"space-c", "space-d", "space-f"},
				}, store.SecurityGroup{
					Guid:              "fourth-guid",
					Name:              "fourth-name",
					Rules:             "fourthRules",
					RunningSpaceGuids: []string{"space-d"},
					StagingSpaceGuids: []string{"space-d"},
				}))
				Expect(pagination.Next).To(Equal(0))
			})
		})

		Context("when a page has a limit", func() {
			It("returns the requested limit", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-e", "space-d"}, store.Page{Limit: 1, From: 3})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(1))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "third-guid",
					Name:              "third-name",
					Rules:             "thirdRules",
					RunningSpaceGuids: []string{"space-c", "space-d", "space-e"},
					StagingSpaceGuids: []string{"space-c", "space-d", "space-f"},
				}))
				Expect(pagination).To(Equal(store.Pagination{Next: 4}))

				securityGroups, pagination, err = securityGroupsStore.BySpaceGuids([]string{"space-e", "space-d"}, store.Page{Limit: 1, From: 4})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(securityGroups)).To(Equal(1))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "fourth-guid",
					Name:              "fourth-name",
					Rules:             "fourthRules",
					RunningSpaceGuids: []string{"space-d"},
					StagingSpaceGuids: []string{"space-d"},
				}))
				Expect(pagination).To(Equal(store.Pagination{Next: 0}))
			})
		})

		Context("when there is a public staging security group", func() {
			BeforeEach(func() {
				securityGroups = []store.SecurityGroup{{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					StagingDefault:    true,
					RunningSpaceGuids: []string{"space-a"},
				}, {
					Guid:              "second-guid",
					Name:              "second-name",
					Rules:             "secondRules",
					RunningSpaceGuids: []string{"space-b"},
					StagingSpaceGuids: []string{"space-b"},
				}, {}}

				err := securityGroupsStore.Replace(securityGroups)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns it even if it is not requested by space guid", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-b"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(2))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					StagingDefault:    true,
					RunningSpaceGuids: []string{"space-a"},
				}, store.SecurityGroup{
					Guid:              "second-guid",
					Name:              "second-name",
					Rules:             "secondRules",
					RunningSpaceGuids: []string{"space-b"},
					StagingSpaceGuids: []string{"space-b"},
				}))
				Expect(pagination.Next).To(Equal(0))

			})

			It("returns it when no space guids are provided", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(1))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					StagingDefault:    true,
					RunningSpaceGuids: []string{"space-a"},
				}))
				Expect(pagination.Next).To(Equal(0))
			})
		})

		FContext("when there is a public running security group", func() {
			BeforeEach(func() {
				securityGroups = []store.SecurityGroup{{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					RunningDefault:    true,
					RunningSpaceGuids: []string{"space-a"},
				}}

				var spaceCache store.SpaceCache
				spaceCache = store.SpaceCache{
					Spaces: map[string]store.Space{
						"space-b": store.Space{
							Guid: "space-b",
							ASGs: map[string]store.SecurityGroup{
								"second-guid": store.SecurityGroup{
									Guid:              "second-guid",
									Name:              "second-name",
									Rules:             "secondRules",
									RunningSpaceGuids: []string{"space-b"},
									StagingSpaceGuids: []string{"space-b"},
								},
							},
						},
					},
				}

				err := securityGroupsStore.Replace(securityGroups)
				Expect(err).ToNot(HaveOccurred())

				err = securityGroupsStore.UpdateSpaceCache(spaceCache)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns it even if it is not requested by space guid", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{"space-b"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(2))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					RunningDefault:    true,
					RunningSpaceGuids: []string{"space-a"},
				}, store.SecurityGroup{
					Guid:              "second-guid",
					Name:              "second-name",
					Rules:             "secondRules",
					RunningSpaceGuids: []string{"space-b"},
					StagingSpaceGuids: []string{"space-b"},
				}))
				Expect(pagination.Next).To(Equal(0))
			})

			It("returns it when no space guids are requested", func() {
				securityGroups, pagination, err := securityGroupsStore.BySpaceGuids([]string{}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(securityGroups)).To(Equal(1))
				Expect(securityGroups).To(ConsistOf(store.SecurityGroup{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					RunningDefault:    true,
					RunningSpaceGuids: []string{"space-a"},
				}))
				Expect(pagination.Next).To(Equal(0))
			})
		})
	})

	Describe("Replace", func() {
		var initialRules, newRules []store.SecurityGroup

		BeforeEach(func() {
			initialRules = []store.SecurityGroup{{
				Guid:              "first-guid",
				Name:              "first-asg",
				Rules:             "firstRules",
				RunningSpaceGuids: []string{"first-space"},
			}, {
				Guid:              "second-guid",
				Name:              "second-name",
				Rules:             "secondRules",
				RunningSpaceGuids: []string{"second-space"},
				StagingSpaceGuids: []string{"second-space"},
			}}

			// Validates that we delete the first guid, update the second guid, add a third in place of the first
			newRules = []store.SecurityGroup{{
				Guid:              "third-guid",
				Name:              "third-name",
				Rules:             "thirdRules",
				StagingSpaceGuids: []string{"third-space"},
				StagingDefault:    true,
				RunningSpaceGuids: []string{},
			}, {
				Guid:              "second-guid",
				Name:              "second-name",
				Rules:             "secondUpdatedRules",
				StagingSpaceGuids: []string{"first-space", "second-space"},
				RunningSpaceGuids: []string{"first-space", "second-space"},
				StagingDefault:    true,
				RunningDefault:    true,
			}}

			err := securityGroupsStore.Replace(initialRules)
			Expect(err).ToNot(HaveOccurred())
		})

		It("updates last updated field", func() {
			lastUpdatedOriginal, err := securityGroupsStore.LastUpdated()
			Expect(err).NotTo(HaveOccurred())
			Expect(lastUpdatedOriginal).NotTo(BeNil())
			time.Sleep(1 * time.Second)

			err = securityGroupsStore.Replace(newRules)
			Expect(err).NotTo(HaveOccurred())

			lastUpdatedNew, err := securityGroupsStore.LastUpdated()
			Expect(err).NotTo(HaveOccurred())
			Expect(lastUpdatedNew).To(BeNumerically(">", lastUpdatedOriginal))
		})

		It("replaces the spaceSecurityGroupsStore data with the newly provided data", func() {
			err := securityGroupsStore.Replace(newRules)
			Expect(err).ToNot(HaveOccurred())

			securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
			Expect(err).ToNot(HaveOccurred())

			Expect(securityGroups).To(ConsistOf(newRules))
		})

		It("works if data is the same", func() {
			err := securityGroupsStore.Replace(initialRules)
			Expect(err).ToNot(HaveOccurred())

			securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
			Expect(err).ToNot(HaveOccurred())

			Expect(securityGroups).To(ConsistOf(initialRules))
		})

		Context("when errors occur", func() {
			var mockDB *fakes.Db
			var tx *dbfakes.Transaction
			BeforeEach(func() {
				mockDB = new(fakes.Db)
				tx = new(dbfakes.Transaction)
				mockDB.BeginxReturns(tx, nil)
				securityGroupsStore.Conn = mockDB
			})

			Context("beginning a transaction", func() {
				BeforeEach(func() {
					mockDB.BeginxReturns(nil, errors.New("can't create a transaction"))
				})

				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("create transaction: can't create a transaction"))
				})
			})

			Context("getting existing security groups", func() {
				BeforeEach(func() {
					tx.QueryxReturns(nil, errors.New("can't exec SQL"))
				})

				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("selecting security groups: can't exec SQL"))
				})

				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("inserting a security group", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(0, nil, errors.New("can't exec SQL"))
				})

				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("saving security group third-guid (third-name): can't exec SQL"))
				})

				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("committing a transaction fails", func() {
				BeforeEach(func() {
					tx.CommitReturns(errors.New("can't commit transaction"))
				})

				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("committing transaction: can't commit transaction"))
				})

				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})
		})
	})

	FDescribe("UpdateSpaceCache()", func() {
		Context("when given a mapping", func() {
			var spaceCache store.SpaceCache
			BeforeEach(func() {
				spaceCache = store.SpaceCache{
					Spaces: map[string]store.Space{
						"space-1": store.Space{
							Guid: "space-1",
							ASGs: map[string]store.SecurityGroup{
								"third-guid": store.SecurityGroup{
									Guid:              "third-guid",
									Name:              "third-name",
									Rules:             "thirdRules",
									StagingSpaceGuids: []string{"space-3"},
									StagingDefault:    true,
									RunningSpaceGuids: []string{},
								},
								"asg-1": store.SecurityGroup{
									Guid:              "asg-1",
									Name:              "asg-1",
									Rules:             "secondRules",
									StagingSpaceGuids: []string{"space-2"},
									RunningSpaceGuids: []string{"space-1"},
								},
								"asg-2": store.SecurityGroup{
									Guid:              "asg-2",
									Name:              "asg-2",
									Rules:             "secondRules",
									StagingSpaceGuids: []string{"space-2"},
									StagingDefault:    false,
									RunningDefault:    true,
									RunningSpaceGuids: []string{"space-1"},
								},
							},
						},
						"space-2": store.Space{Guid: "space-2"},
						"space-3": store.Space{Guid: "space-3"},
					},
				}
			})

			It("populates the database", func() {
				err := securityGroupsStore.UpdateSpaceCache(spaceCache)
				Expect(err).NotTo(HaveOccurred())
				rows, err := realDb.DB.Query("SELECT guid, asgs, lastUpdated, hash  FROM spaces ORDER BY id")
				Expect(err).ToNot(HaveOccurred())

				var actualSpaceCache store.SpaceCache
				actualSpaceCache.Spaces = map[string]store.Space{}
				for rows.Next() {
					var guid, hash string
					var asgDefinitions store.SecurityGroups
					var lastUpdated time.Time
					err := rows.Scan(&guid, &asgDefinitions, &lastUpdated, &hash)
					Expect(err).NotTo(HaveOccurred())
					Expect(err).NotTo(HaveOccurred())
					actualSpaceCache.Spaces[guid] = store.Space{
						Guid: guid,
					}
				}

				Expect(actualSpaceCache).To(Equal(store.SpaceCache{
					Spaces: map[string]store.Space{
						"space-1": store.Space{Guid: "space-1"},
						"space-2": store.Space{Guid: "space-2"},
						"space-3": store.Space{Guid: "space-3"},
					}}))
			})
		})
	})

	FDescribe("CheckForASGUpdates()", func() {
		var spaceCache store.SpaceCache
		BeforeEach(func() {
			spaceCache = store.SpaceCache{
				Spaces: map[string]store.Space{
					"space-1": store.Space{
						Guid: "space-1",
						ASGs: map[string]store.SecurityGroup{
							"third-guid": store.SecurityGroup{
								Guid:              "third-guid",
								Name:              "third-name",
								Rules:             "thirdRules",
								StagingSpaceGuids: []string{"space-3"},
								StagingDefault:    true,
								RunningSpaceGuids: []string{},
							},
							"asg-1": store.SecurityGroup{
								Guid:              "asg-1",
								Name:              "asg-1",
								Rules:             "secondRules",
								StagingSpaceGuids: []string{"space-2"},
								RunningSpaceGuids: []string{"space-1"},
							},
							"asg-2": store.SecurityGroup{
								Guid:              "asg-2",
								Name:              "asg-2",
								Rules:             "secondRules",
								StagingSpaceGuids: []string{"space-2"},
								StagingDefault:    false,
								RunningDefault:    true,
								RunningSpaceGuids: []string{"space-1"},
							},
						},
					},
					"space-2": store.Space{Guid: "space-2"},
					"space-3": store.Space{Guid: "space-3"},
				},
			}

			migrateAndPopulateTags(realDb, 1)

			err := securityGroupsStore.UpdateSpaceCache(spaceCache)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(1 * time.Second)
			spaceCache.Spaces["space-3"] = store.Space{
				ASGs: map[string]store.SecurityGroup{
					"asg-2": store.SecurityGroup{
						Guid:              "asg-2",
						Name:              "asg-2",
						Rules:             "secondRules",
						StagingSpaceGuids: []string{"space-1"},
						RunningSpaceGuids: []string{"space-3"},
					}}}
			err = securityGroupsStore.UpdateSpaceCache(spaceCache)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when global ASGs have been updated since the last check", func() {
			BeforeEach(func() {
				_, err := securityGroupsStore.Conn.Query("UPDATE security_groups_info SET last_updated=CURRENT_TIMESTAMP(6)")
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns true", func() {
				wasUpdated, err := securityGroupsStore.CheckForASGUpdates([]string{}, time.Now().Add(-1*time.Second))
				Expect(err).ToNot(HaveOccurred())
				Expect(wasUpdated).To(BeTrue())
			})
		})

		Context("when the global ASGs have not been updated since the last check", func() {
			BeforeEach(func() {
				_, err := securityGroupsStore.Conn.Query(`UPDATE security_groups_info SET last_updated = '1970-01-01 00:00:01.000000'`)
				Expect(err).NotTo(HaveOccurred())
			})
			Context("when given no security groups", func() {
				Context("and the last check is newer than the last update", func() {
					It("returns false", func() {
						wasUpdated, err := securityGroupsStore.CheckForASGUpdates([]string{}, time.Now().Add(1*time.Second))
						Expect(err).ToNot(HaveOccurred())
						Expect(wasUpdated).To(BeFalse())
					})
				})
				Context("and the last check time is older than the last update", func() {
					It("returns true", func() {
						wasUpdated, err := securityGroupsStore.CheckForASGUpdates([]string{}, time.Time{})
						Expect(err).ToNot(HaveOccurred())
						Expect(wasUpdated).To(BeTrue())
					})
				})
			})
			Context("when given security groups", func() {
				Context("and the last check is newer than the last update", func() {
					It("returns false", func() {
						wasUpdated, err := securityGroupsStore.CheckForASGUpdates([]string{"space-2"}, time.Now().Add(1*time.Second))
						Expect(err).ToNot(HaveOccurred())
						Expect(wasUpdated).To(BeFalse())
					})
				})
				Context("and the last check time is older than the last update", func() {
					It("returns true", func() {
						wasUpdated, err := securityGroupsStore.CheckForASGUpdates([]string{"space-2"}, time.Time{})
						Expect(err).ToNot(HaveOccurred())
						Expect(wasUpdated).To(BeTrue())
					})
				})
			})
			// FIXME: sad path tests
			// FIXME: test that an update to an unrelated sg doesn't trigger true
		})
	})

	Describe("LastUpdated()", func() {
		var currentTime int64
		BeforeEach(func() {

			migrateAndPopulateTags(realDb, 1)
			currentTime = time.Now().UnixNano()

			securityGroupsStore.Replace([]store.SecurityGroup{{
				Guid:              "third-guid",
				Name:              "third-name",
				Rules:             "thirdRules",
				StagingSpaceGuids: []string{"third-space"},
				StagingDefault:    true,
				RunningSpaceGuids: []string{},
			}})
		})
		It("returns a timestamp in UnixNano format", func() {
			updatedTime, err := securityGroupsStore.LastUpdated()
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTime).To(BeNumerically(">", currentTime))
		})
	})

})
