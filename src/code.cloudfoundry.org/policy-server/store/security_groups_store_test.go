package store_test

import (
	"errors"
	"fmt"
	"time"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/store/fakes"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo"
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

		logger := lager.NewLogger("Security Groups Store Test")

		var err error
		realDb, err = dbHelper.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Security Groups Store Test", "Security Groups Store Test", logger)
		Expect(err).NotTo(HaveOccurred())
		securityGroupsStore = &store.SGStore{
			Conn: realDb,
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
		})

		Context("when there is a public running security group", func() {
			BeforeEach(func() {
				securityGroups = []store.SecurityGroup{{
					Guid:              "first-guid",
					Name:              "first-asg",
					Rules:             "firstRules",
					RunningDefault:    true,
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
})
