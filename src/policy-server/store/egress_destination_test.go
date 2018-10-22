package store_test

import (
	"errors"
	"fmt"
	"policy-server/store"
	"test-helpers"
	"time"

	"github.com/nu7hatch/gouuid"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressDestination", func() {
	Context("when using a real database", func() {
		var (
			dbConf dbHelper.Config
			realDb *dbHelper.ConnWrapper

			terminalsTable         *store.TerminalsTable
			egressDestinationTable *store.EgressDestinationTable

			terminalIds []string
			tx          dbHelper.Transaction
			err         error
		)

		BeforeEach(func() {
			terminalIds = []string{}

			dbConf = testsupport.GetDBConfig()
			dbConf.DatabaseName = fmt.Sprintf("egress_destination_test_node_%d", time.Now().UnixNano())
			dbConf.Timeout = 30
			testhelpers.CreateDatabase(dbConf)

			logger := lager.NewLogger("Egress Destination Test")

			var err error
			realDb, err = dbHelper.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Egress Destination Test", "Egress Destination Test", logger)
			Expect(err).NotTo(HaveOccurred())

			migrate(realDb)

			egressDestinationTable = &store.EgressDestinationTable{}

			terminalsTable = &store.TerminalsTable{
				Guids: &store.GuidGenerator{},
			}

			tx, err = realDb.Beginx()
			Expect(err).NotTo(HaveOccurred())

			terminalId, err := terminalsTable.Create(tx)
			Expect(err).NotTo(HaveOccurred())
			_, err = egressDestinationTable.CreateIPRange(tx, terminalId, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, -1, -1)
			Expect(err).NotTo(HaveOccurred())
			terminalIds = append(terminalIds, terminalId)

			terminalId, err = terminalsTable.Create(tx)
			Expect(err).NotTo(HaveOccurred())
			_, err = egressDestinationTable.CreateIPRange(tx, terminalId, "1.1.1.2", "2.2.2.3", "udp", 8082, 8083, 7, 8)
			Expect(err).NotTo(HaveOccurred())
			terminalIds = append(terminalIds, terminalId)
		})

		AfterEach(func() {
			err = tx.Rollback()
			Expect(err).NotTo(HaveOccurred())

			if realDb != nil {
				Expect(realDb.Close()).To(Succeed())
			}
			testhelpers.RemoveDatabase(dbConf)
		})

		Context("when a destination metadata doesn't exist for destination", func() {
			It("returns empty strings for name/description", func() {
				By("All")

				destinations, err := egressDestinationTable.All(tx)
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(2))

				_, err = uuid.ParseHex(destinations[0].GUID)
				Expect(err).NotTo(HaveOccurred())

				Expect(destinations[0].GUID).To(Equal(terminalIds[0]))
				Expect(destinations[0].Name).To(Equal(""))
				Expect(destinations[0].Description).To(Equal(""))
				Expect(destinations[0].Protocol).To(Equal("tcp"))
				Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
				Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
				Expect(destinations[0].ICMPType).To(Equal(-1))
				Expect(destinations[0].ICMPCode).To(Equal(-1))

				_, err = uuid.ParseHex(destinations[1].GUID)
				Expect(err).NotTo(HaveOccurred())

				Expect(destinations[1].GUID).To(Equal(terminalIds[1]))
				Expect(destinations[1].Name).To(Equal(""))
				Expect(destinations[1].Description).To(Equal(""))
				Expect(destinations[1].Protocol).To(Equal("udp"))
				Expect(destinations[1].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.2", End: "2.2.2.3"}}))
				Expect(destinations[1].Ports).To(Equal([]store.Ports{{Start: 8082, End: 8083}}))
				Expect(destinations[1].ICMPType).To(Equal(7))
				Expect(destinations[1].ICMPCode).To(Equal(8))

				By("GetByGUID")
				destinations, err = egressDestinationTable.GetByGUID(tx, terminalIds...)
				Expect(err).NotTo(HaveOccurred())

				Expect(destinations).To(HaveLen(2))

				_, err = uuid.ParseHex(destinations[0].GUID)
				Expect(err).NotTo(HaveOccurred())

				Expect(destinations[0].GUID).To(Equal(terminalIds[0]))
				Expect(destinations[0].Name).To(Equal(""))
				Expect(destinations[0].Description).To(Equal(""))
				Expect(destinations[0].Protocol).To(Equal("tcp"))
				Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
				Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
				Expect(destinations[0].ICMPType).To(Equal(-1))
				Expect(destinations[0].ICMPCode).To(Equal(-1))

				_, err = uuid.ParseHex(destinations[1].GUID)
				Expect(err).NotTo(HaveOccurred())

				Expect(destinations[1].GUID).To(Equal(terminalIds[1]))
				Expect(destinations[1].Name).To(Equal(""))
				Expect(destinations[1].Description).To(Equal(""))
				Expect(destinations[1].Protocol).To(Equal("udp"))
				Expect(destinations[1].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.2", End: "2.2.2.3"}}))
				Expect(destinations[1].Ports).To(Equal([]store.Ports{{Start: 8082, End: 8083}}))
				Expect(destinations[1].ICMPType).To(Equal(7))
				Expect(destinations[1].ICMPCode).To(Equal(8))

				_, err = egressDestinationTable.GetByGUID(tx, "garbage id")
				Expect(err).NotTo(HaveOccurred())

				By("Delete")
				err = egressDestinationTable.Delete(tx, "garbage id")
				Expect(err).ToNot(HaveOccurred())

				err = egressDestinationTable.Delete(tx, terminalIds[0])
				Expect(err).NotTo(HaveOccurred())

				destinations, err = egressDestinationTable.All(tx)
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(1))
				Expect(destinations[0].GUID).To(Equal(terminalIds[1]))
			})
		})

		Context("when a destination metadata exist for destination", func() {
			BeforeEach(func() {
				metadataTable := store.DestinationMetadataTable{}
				err = metadataTable.Upsert(tx, terminalIds[0], "dest name", "dest desc")
				Expect(err).NotTo(HaveOccurred())
			})

			Context("GetByGUID", func() {
				It("returns the name/description", func() {
					By("All")

					destinations, err := egressDestinationTable.All(tx)
					Expect(err).NotTo(HaveOccurred())

					_, err = uuid.ParseHex(destinations[0].GUID)
					Expect(err).NotTo(HaveOccurred())

					Expect(destinations[0].GUID).To(Equal(terminalIds[0]))
					Expect(destinations[0].Name).To(Equal("dest name"))
					Expect(destinations[0].Description).To(Equal("dest desc"))
					Expect(destinations[0].Protocol).To(Equal("tcp"))
					Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
					Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
					Expect(destinations[0].ICMPType).To(Equal(-1))
					Expect(destinations[0].ICMPCode).To(Equal(-1))

					By("GetByGUID")

					destinations, err = egressDestinationTable.GetByGUID(tx, terminalIds[0])
					Expect(err).NotTo(HaveOccurred())

					Expect(destinations).To(HaveLen(1))

					_, err = uuid.ParseHex(destinations[0].GUID)
					Expect(err).NotTo(HaveOccurred())

					Expect(destinations[0].GUID).To(Equal(terminalIds[0]))
					Expect(destinations[0].Name).To(Equal("dest name"))
					Expect(destinations[0].Description).To(Equal("dest desc"))
					Expect(destinations[0].Protocol).To(Equal("tcp"))
					Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
					Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
					Expect(destinations[0].ICMPType).To(Equal(-1))
					Expect(destinations[0].ICMPCode).To(Equal(-1))
				})
			})
		})
	})

	Context("edge cases with fake database", func() {
		var (
			tx                     *dbfakes.Transaction
			egressDestinationTable *store.EgressDestinationTable
		)
		BeforeEach(func() {
			tx = new(dbfakes.Transaction)
			egressDestinationTable = &store.EgressDestinationTable{}
		})
		Context("when the transaction returns an error", func() {
			BeforeEach(func() {
				tx.ExecReturns(nil, errors.New("bad things happened"))
			})
			It("returns the error", func() {
				Expect(egressDestinationTable.UpdateIPRange(tx, "", "", "", "", int64(3), int64(4), int64(5), int64(6))).To(MatchError("bad things happened"))
			})
		})

		Context("update", func() {
			It("passes an error from Exec if Exec fails", func() {
				tx.ExecReturns(nil, errors.New("bigger error"))
				err := egressDestinationTable.UpdateIPRange(tx, "", "", "", "", int64(3), int64(4), int64(5), int64(6))
				Expect(err).To(MatchError("bigger error"))
			})
		})

		Context("GetByName", func() {
			Context("when there is an error running the query", func() {
				BeforeEach(func() {
					tx.QueryxReturns(nil, errors.New("error with transaction"))
				})

				It("returns an error", func() {
					_, err := egressDestinationTable.GetByName(tx, "some-name")
					Expect(err).To(MatchError("running query: error with transaction"))
				})
			})
		})
	})

})
