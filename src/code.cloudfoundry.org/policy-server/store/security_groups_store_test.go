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
		securityGroupsStore    *store.SecurityGroupsStore
		dbConf                 dbHelper.Config
		realDb                 *dbHelper.ConnWrapper
		initialRules, newRules []store.SecurityGroup
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
	})

	AfterEach(func() {
		Expect(realDb.Close()).To(Succeed())
		testhelpers.RemoveDatabase(dbConf)
	})

	Describe("Replace", func() {
		BeforeEach(func() {
			err := securityGroupsStore.Replace(initialRules)
			Expect(err).ToNot(HaveOccurred())
		})
		It("replaces the spaceSecurityGroupsStore data with the newly provided data", func() {
			err := securityGroupsStore.Replace(newRules)
			Expect(err).ToNot(HaveOccurred())
			securityGroups := map[string]*store.SecurityGroup{}
			stagingGuids := []string{}
			runningGuids := []string{}
			rows, err := realDb.DB.Query(`SELECT guid, name, rules, running_default, staging_default FROM security_groups`)
			Expect(err).ToNot(HaveOccurred())
			for rows.Next() {
				var guid, name, rules string
				var runningDefault, stagingDefault bool
				err = rows.Scan(&guid, &name, &rules, &runningDefault, &stagingDefault)
				Expect(err).ToNot(HaveOccurred())
				securityGroups[guid] = &store.SecurityGroup{
					Guid:              guid,
					Name:              name,
					Rules:             rules,
					StagingDefault:    stagingDefault,
					StagingSpaceGuids: stagingGuids,
					RunningDefault:    runningDefault,
					RunningSpaceGuids: runningGuids,
				}
			}
			rows.Close()

			rows, err = realDb.DB.Query(`SELECT space_guid, security_group_guid FROM staging_security_groups_spaces`)
			Expect(err).ToNot(HaveOccurred())
			for rows.Next() {
				var spaceGuid, securityGroupGuid string
				err := rows.Scan(&spaceGuid, &securityGroupGuid)
				Expect(err).ToNot(HaveOccurred())
				securityGroups[securityGroupGuid].StagingSpaceGuids = append(
					securityGroups[securityGroupGuid].StagingSpaceGuids, spaceGuid)
			}
			rows.Close()
			rows, err = realDb.DB.Query(`SELECT space_guid, security_group_guid FROM running_security_groups_spaces`)
			Expect(err).ToNot(HaveOccurred())
			for rows.Next() {
				var spaceGuid, securityGroupGuid string
				err := rows.Scan(&spaceGuid, &securityGroupGuid)
				Expect(err).ToNot(HaveOccurred())
				securityGroups[securityGroupGuid].RunningSpaceGuids = append(
					securityGroups[securityGroupGuid].RunningSpaceGuids, spaceGuid)
			}
			rows.Close()
			sgList := []store.SecurityGroup{}
			for _, sg := range securityGroups {
				sgList = append(sgList, *sg)
			}

			Expect(sgList).To(ConsistOf(newRules))
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

			Context("deleting data", func() {
				BeforeEach(func() {
					tx.ExecReturns(nil, errors.New("can't exec SQL"))
				})
				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("deleting running security group associations: can't exec SQL"))
				})
				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("inserting a security group", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(3, nil, errors.New("can't exec SQL"))
				})
				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("adding new security group third-guid (third-name): can't exec SQL"))
				})
				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("associating a space with a staging security group", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(4, nil, errors.New("can't exec SQL"))
				})
				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("associating staging security group third-guid (third-name) to space third-space: can't exec SQL"))
				})
				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("associating a space with a running security group", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(8, nil, errors.New("can't exec SQL"))
				})
				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("associating running security group second-guid (second-name) to space first-space: can't exec SQL"))
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
