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
	"code.cloudfoundry.org/policy-server/store/helpers"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("SecurityGroupsStore", func() {
	var (
		securityGroupsStore *store.SGStore
		dbConf              dbHelper.Config
		realDb              *dbHelper.ConnWrapper
		testLogger          *lagertest.TestLogger
	)

	getNumRecords := func(table string) int {
		var count int
		ExpectWithOffset(1, realDb).ToNot(BeNil())
		err := realDb.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return count
	}

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("security_groups_store_test_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		var err error
		testLogger = lagertest.NewTestLogger("asg-syncer-test")
		realDb, err = dbHelper.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Security Groups Store Test", "Security Groups Store Test", testLogger)
		Expect(err).NotTo(HaveOccurred())
		securityGroupsStore = &store.SGStore{
			Conn:   realDb,
			Logger: testLogger,
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
			It("fetches asgs attached to provided spaces", func() {
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
			It("fetches attached to provided spaces", func() {
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
		var initialRules, newRules, emptyRules []store.SecurityGroup

		BeforeEach(func() {
			initialRules = []store.SecurityGroup{{
				Guid:              "first-guid",
				Name:              "first-asg",
				Rules:             "firstRules",
				RunningSpaceGuids: []string{"first-space"},
				StagingSpaceGuids: []string{"fourth-space"},
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

			emptyRules = []store.SecurityGroup{}

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
		Context("when the only change is a deletion", func() {
			BeforeEach(func() {
				newRules = initialRules[0:1]
			})
			It("deletes the record", func() {
				err := securityGroupsStore.Replace(newRules)
				Expect(err).ToNot(HaveOccurred())

				securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(securityGroups).To(ConsistOf(newRules))
			})
		})
		Context("when two ASGs change but the third doesn't", func() {
			var hashes []string
			BeforeEach(func() {
				newRules = initialRules
				newRules[1].Guid = "new-guid"
				newRules = append(newRules, store.SecurityGroup{
					Guid:              "third-guid",
					Name:              "third-name",
					Rules:             "thirdRules",
					StagingSpaceGuids: []string{"third-space"},
					StagingDefault:    true,
					RunningSpaceGuids: []string{},
				})
				Eventually(testLogger).Should(gbytes.Say("committing-transaction"))

				rows, err := realDb.Query("SELECT hash FROM security_groups ORDER BY id")
				Expect(err).NotTo(HaveOccurred())
				for rows.Next() {
					var hash string
					err := rows.Scan(&hash)
					Expect(err).NotTo(HaveOccurred())
					hashes = append(hashes, hash)
				}
			})
			It("only updates the two changing", func() {
				err := securityGroupsStore.Replace(newRules)
				Expect(err).ToNot(HaveOccurred())

				securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())
				Expect(securityGroups).To(ConsistOf(newRules))

				Eventually(testLogger).Should(gbytes.Say(`"num_records":2`))

				var newHashes []string
				rows, err := realDb.Query("SELECT hash FROM security_groups ORDER BY id")
				Expect(err).NotTo(HaveOccurred())
				for rows.Next() {
					var hash string
					err := rows.Scan(&hash)
					Expect(err).NotTo(HaveOccurred())
					newHashes = append(newHashes, hash)
				}
				Expect(newHashes[0]).To(Equal(hashes[0]))
				Expect(newHashes[1]).To(Equal("f842efd3fd4944dc6dd669b8f6817f914826485ebec5aa697632d8a10f3ddc30"))
				Expect(newHashes[2]).To(Equal("2a3d6609af1920ccf8fa2c9d2e5d829592ee91f146c46fee2506c01a59aacd3f"))
			})
		})
		Context("testing security-group-space associations", func() {
			BeforeEach(func() {

				newRules[0].StagingDefault = false
				newRules[0].RunningDefault = false
				newRules[1].StagingDefault = false
				newRules[1].RunningDefault = false
			})
			Context("when a security group loses a running space binding", func() {
				BeforeEach(func() {
					newRules[1].RunningSpaceGuids = []string{"third-space", "first-space"}
				})
				It("removes the association for that space from that ASG", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).ToNot(HaveOccurred())

					results := map[string][]string{}
					query := helpers.RebindForSQLDialect("SELECT security_group_guid, space_guid FROM running_security_groups_spaces WHERE security_group_guid = ? ORDER BY space_guid", securityGroupsStore.Conn.DriverName())
					rows, err := securityGroupsStore.Conn.Query(query, "second-guid")
					Expect(err).NotTo(HaveOccurred())
					for rows.Next() {
						var sg, space string
						err := rows.Scan(&sg, &space)
						Expect(err).NotTo(HaveOccurred())

						results[sg] = append(results[sg], space)
					}
					Expect(results).To(Equal(map[string][]string{
						"second-guid": []string{"first-space", "third-space"},
					}))
				})
			})
			Context("when a security group loses a staging space binding", func() {
				BeforeEach(func() {
					newRules[1].StagingSpaceGuids = []string{"third-space", "first-space"}
				})
				It("removes the association for that space from that ASG", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).ToNot(HaveOccurred())

					results := map[string][]string{}
					query := helpers.RebindForSQLDialect("SELECT security_group_guid, space_guid FROM staging_security_groups_spaces WHERE security_group_guid = ? ORDER BY space_guid", securityGroupsStore.Conn.DriverName())
					rows, err := securityGroupsStore.Conn.Query(query, "second-guid")
					Expect(err).NotTo(HaveOccurred())
					for rows.Next() {
						var sg, space string
						err := rows.Scan(&sg, &space)
						Expect(err).NotTo(HaveOccurred())

						results[sg] = append(results[sg], space)
					}
					Expect(results).To(Equal(map[string][]string{
						"second-guid": []string{"first-space", "third-space"},
					}))
				})
			})
			Context("when a security group loses its staging space bindings", func() {
				BeforeEach(func() {
					newRules[1].StagingSpaceGuids = []string{}
				})
				It("removes all associations for that ASG", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).ToNot(HaveOccurred())

					var associations int
					query := helpers.RebindForSQLDialect("SELECT COUNT(*) FROM staging_security_groups_spaces WHERE security_group_guid = ?", securityGroupsStore.Conn.DriverName())
					err = securityGroupsStore.Conn.QueryRow(query, "second-guid").Scan(&associations)
					Expect(err).NotTo(HaveOccurred())
					Expect(associations).To(Equal(0))

					err = securityGroupsStore.Conn.QueryRow("SELECT COUNT(*) FROM staging_security_groups_spaces").Scan(&associations)
					Expect(err).NotTo(HaveOccurred())
					Expect(associations).ToNot(Equal(0))
				})
			})

			Context("when a security group loses its running space bindings", func() {
				BeforeEach(func() {
					newRules[1].RunningSpaceGuids = []string{}
					newRules[0].RunningSpaceGuids = []string{"fourth-space"}
				})
				It("removes all associations from that ASG", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).ToNot(HaveOccurred())

					var associations int
					query := helpers.RebindForSQLDialect("SELECT COUNT(*) FROM running_security_groups_spaces WHERE security_group_guid = ?", securityGroupsStore.Conn.DriverName())
					err = securityGroupsStore.Conn.QueryRow(query, "second-guid").Scan(&associations)
					Expect(err).NotTo(HaveOccurred())
					Expect(associations).To(Equal(0))

					err = securityGroupsStore.Conn.QueryRow("SELECT COUNT(*) FROM running_security_groups_spaces").Scan(&associations)
					Expect(err).NotTo(HaveOccurred())
					Expect(associations).ToNot(Equal(0))
				})
			})
			Context("when no more running asgs are directly bound", func() {
				BeforeEach(func() {
					newRules[0].RunningSpaceGuids = []string{}
					newRules[1].RunningSpaceGuids = []string{}
				})
				It("removes all space associations from from all ASGs", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).ToNot(HaveOccurred())

					var associations int
					err = securityGroupsStore.Conn.QueryRow("SELECT COUNT(*) FROM running_security_groups_spaces").Scan(&associations)
					Expect(err).NotTo(HaveOccurred())
					Expect(associations).To(Equal(0))
				})
			})
			Context("when no more staging asgs are directly bound", func() {
				BeforeEach(func() {
					newRules[0].StagingSpaceGuids = []string{}
					newRules[1].StagingSpaceGuids = []string{}
				})
				It("removes all space associations from from all ASGs", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).ToNot(HaveOccurred())

					var associations int
					err = securityGroupsStore.Conn.QueryRow("SELECT COUNT(*) FROM staging_security_groups_spaces").Scan(&associations)
					Expect(err).NotTo(HaveOccurred())
					Expect(associations).To(Equal(0))
				})
			})
			Context("when the only updated ASGs are ones that don't have space bindings", func() {
				BeforeEach(func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).ToNot(HaveOccurred())
					securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
					Expect(err).ToNot(HaveOccurred())
					Expect(securityGroups).To(ConsistOf(newRules))

					newRules = append(newRules, store.SecurityGroup{
						Name:           "only-global",
						Guid:           "only-global-guid",
						StagingDefault: true,
						RunningDefault: true,
					})
				})
				It("doesn't affect any of the space bindings", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).NotTo(HaveOccurred())

					securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
					Expect(err).ToNot(HaveOccurred())

					Expect(securityGroups).To(ConsistOf(newRules))

				})
			})
		})
		Context("when no more asgs exist", func() {
			It("removes all ASGs", func() {
				Expect(getNumRecords("security_groups")).ToNot(Equal(0))
				Expect(getNumRecords("staging_security_groups_spaces")).ToNot(Equal(0))
				Expect(getNumRecords("running_security_groups_spaces")).ToNot(Equal(0))

				err := securityGroupsStore.Replace(emptyRules)
				Expect(err).ToNot(HaveOccurred())

				securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(securityGroups).To(ConsistOf(emptyRules))
				time.Sleep(1 * time.Second)
				Expect(getNumRecords("security_groups")).To(Equal(0))
				Expect(getNumRecords("staging_security_groups_spaces")).To(Equal(0))
				Expect(getNumRecords("running_security_groups_spaces")).To(Equal(0))
			})
		})
		Context("when no pre-existing data exists", func() {
			BeforeEach(func() {
				err := securityGroupsStore.Replace(emptyRules)
				Expect(err).NotTo(HaveOccurred())

				Expect(getNumRecords("security_groups")).To(Equal(0))
				Expect(getNumRecords("staging_security_groups_spaces")).To(Equal(0))
				Expect(getNumRecords("running_security_groups_spaces")).To(Equal(0))
			})
			It("creates new data", func() {
				err := securityGroupsStore.Replace(newRules)
				Expect(err).ToNot(HaveOccurred())

				securityGroups, _, err := securityGroupsStore.BySpaceGuids([]string{"first-space", "second-space", "third-space"}, store.Page{})
				Expect(err).ToNot(HaveOccurred())

				Expect(securityGroups).To(ConsistOf(newRules))

			})
		})

		Context("when errors occur", func() {
			var mockDB *fakes.Db
			var tx *dbfakes.Transaction
			BeforeEach(func() {
				mockDB = new(fakes.Db)
				tx = new(dbfakes.Transaction)
				mockDB.BeginxReturns(tx, nil)
				securityGroupsStore.Conn = mockDB
				mockDB.DriverNameReturns("mysql")
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

			Context("when deleting all ASGs", func() {
				Context("and the error is when deleting ASGs", func() {
					BeforeEach(func() {
						tx.ExecReturnsOnCall(0, nil, errors.New("can't exec SQL"))
					})

					It("returns an error", func() {
						err := securityGroupsStore.Replace(emptyRules)
						Expect(err).To(MatchError("deleting ALL security groups: can't exec SQL"))
					})

					It("rolls back the transaction", func() {
						securityGroupsStore.Replace(newRules)
						Expect(tx.RollbackCallCount()).To(Equal(1))
					})
				})
				Context("and committing the transaction fails", func() {
					BeforeEach(func() {
						tx.CommitReturns(errors.New("can't commit transaction"))
					})

					It("returns an error", func() {
						err := securityGroupsStore.Replace(emptyRules)
						Expect(err).To(MatchError("committing transaction to delete ALL security groups: can't commit transaction"))
					})

					It("rolls back the transaction", func() {
						securityGroupsStore.Replace(newRules)
						Expect(tx.RollbackCallCount()).To(Equal(1))
					})
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
					Expect(err).To(MatchError("upserting security groups: executing batched statement: can't exec SQL"))
				})

				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("updating running security group bindings", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(3, nil, errors.New("can't exec SQL"))
				})

				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("replacing running space associations: deleting previous associations: executing batched statement: can't exec SQL"))
				})

				It("rolls back the transaction", func() {
					securityGroupsStore.Replace(newRules)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})
			Context("updating staging security group bindings", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(1, nil, errors.New("can't exec SQL"))
				})

				It("returns an error", func() {
					err := securityGroupsStore.Replace(newRules)
					Expect(err).To(MatchError("replacing staging space associations: deleting previous associations: executing batched statement: can't exec SQL"))
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

	Describe("ReplaceSecurityGroupSpaceAssociations()", func() {
		var tx *dbfakes.Transaction
		var sgSpaceBindings map[string][]string
		BeforeEach(func() {
			mockDB := new(fakes.Db)
			tx = new(dbfakes.Transaction)
			mockDB.BeginxReturns(tx, nil)
			securityGroupsStore.Conn = mockDB
			mockDB.DriverNameReturns("mysql")

			sgSpaceBindings = map[string][]string{
				"sg-1": {"space-1", "space-2"},
				"sg-2": {"space-2", "space-3"},
			}
		})
		Context("when errors occur", func() {
			Context("during the delete", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(0, nil, errors.New("injected failure"))
				})
				It("returns an error", func() {
					err := securityGroupsStore.ReplaceSecurityGroupSpaceAssociations(tx, "staging_security_groups_spaces", sgSpaceBindings)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("deleting previous associations: executing batched statement: injected failure"))
				})
			})
			Context("during the insert", func() {
				BeforeEach(func() {
					tx.ExecReturnsOnCall(1, nil, errors.New("injected failure"))
				})
				It("returns an error", func() {
					err := securityGroupsStore.ReplaceSecurityGroupSpaceAssociations(tx, "staging_security_groups_spaces", sgSpaceBindings)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("creating new associations: executing batched statement: injected failure"))
				})
			})
		})
		//Happy path tests are all included in tests of Replace() since thats the functionality
		//we actually care about succeeding
	})

	Describe("BatchPreparedStatement()", func() {
		paramsPerRecord := 2
		// we use this many to validate we adhere to the limits of postgres + mysql, rather
		// than testing a smaller dataset against an arbitrarily lowered max to only test batching
		var values []any
		var tx dbHelper.Transaction
		BeforeEach(func() {
			values = []any{}
			for i := range 35000 {
				values = append(values, fmt.Sprintf("fake-asg-%d", i))
				values = append(values, fmt.Sprintf("fake-name-%d", i))
			}
		})
		JustBeforeEach(func() {
			var err error
			tx, err = securityGroupsStore.Conn.Beginx()
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			_, err := realDb.Exec("DELETE FROM staging_security_groups_spaces")
			Expect(err).NotTo(HaveOccurred())
		})

		It("doesn't error and inserts every record", func() {
			err := securityGroupsStore.BatchPreparedStatement(tx, "INSERT INTO security_groups (guid, name) VALUES", "", values, paramsPerRecord)
			Expect(err).NotTo(HaveOccurred())
			Expect(tx.Commit()).To(Succeed())
			rows, err := securityGroupsStore.Conn.Query("SELECT guid, name FROM security_groups ORDER BY guid")
			Expect(err).NotTo(HaveOccurred())
			results := map[string]string{}
			for rows.Next() {
				var sg, name string
				err := rows.Scan(&sg, &name)
				Expect(err).NotTo(HaveOccurred())
				results[sg] = name
			}
			for i := range 35000 {
				space, ok := results[fmt.Sprintf("fake-asg-%d", i)]
				Expect(ok).To(BeTrue())
				Expect(space).To(Equal(fmt.Sprintf("fake-name-%d", i)))
			}
		})
		Context("when run against a mockDB", func() {
			var fakeTx *dbfakes.Transaction
			BeforeEach(func() {
				mockDB := new(fakes.Db)
				fakeTx = new(dbfakes.Transaction)
				mockDB.BeginxReturns(fakeTx, nil)
				securityGroupsStore.Conn = mockDB
				mockDB.DriverNameReturns("mysql")
			})
			It("batches into two chunks", func() {
				err := securityGroupsStore.BatchPreparedStatement(tx, "INSERT INTO security_groups (guid, name) VALUES", "", values, paramsPerRecord)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeTx.ExecCallCount()).To(Equal(2))
			})
			Context("when execution fails", func() {
				BeforeEach(func() {
					fakeTx.ExecReturns(nil, fmt.Errorf("injected failure"))
				})
				It("returns an error", func() {
					err := securityGroupsStore.BatchPreparedStatement(tx, "INSERT INTO security_groups (guid, name) VALUES", "", values, paramsPerRecord)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("executing batched statement: injected failure"))
				})
			})
		})
	})

	Describe("LastUpdated()", func() {
		var currentTime int64
		BeforeEach(func() {

			migrateAndPopulateTags(realDb, 1)
			currentTime = time.Now().UnixNano()

			err := securityGroupsStore.Replace([]store.SecurityGroup{{
				Guid:              "third-guid",
				Name:              "third-name",
				Rules:             "thirdRules",
				StagingSpaceGuids: []string{"third-space"},
				StagingDefault:    true,
				RunningSpaceGuids: []string{},
			}})
			Expect(err).NotTo(HaveOccurred())
		})
		It("returns a timestamp in UnixNano format", func() {
			updatedTime, err := securityGroupsStore.LastUpdated()
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTime).To(BeNumerically(">", currentTime))
		})
	})

})
