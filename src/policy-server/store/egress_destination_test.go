package store_test

import (
	"fmt"
	"policy-server/db"
	"policy-server/store"
	"test-helpers"
	"time"

	"github.com/nu7hatch/gouuid"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressDestination", func() {
	var (
		dbConf dbHelper.Config
		realDb *dbHelper.ConnWrapper

		terminalsTable         *store.TerminalsTable
		egressDestinationTable *store.EgressDestinationTable

		terminalId string
		tx         dbHelper.Transaction
		err        error
	)

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("egress_destination_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Egress Destination Test")
		realDb = db.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Egress Destination Test", "Egress Destination Test", logger)

		migrate(realDb)

		egressDestinationTable = &store.EgressDestinationTable{}

		terminalsTable = &store.TerminalsTable{
			Guids: &store.GuidGenerator{},
		}

		tx, err = realDb.Beginx()
		Expect(err).NotTo(HaveOccurred())

		terminalId, err = terminalsTable.Create(tx)
		Expect(err).NotTo(HaveOccurred())

		_, err = egressDestinationTable.CreateIPRange(tx, terminalId, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, -1, -1)
		Expect(err).NotTo(HaveOccurred())

		err = tx.Commit()
		Expect(err).NotTo(HaveOccurred())

		tx, err = realDb.Beginx()
		Expect(err).NotTo(HaveOccurred())
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

			_, err = uuid.ParseHex(destinations[0].GUID)
			Expect(err).NotTo(HaveOccurred())

			Expect(destinations[0].GUID).To(Equal(terminalId))
			Expect(destinations[0].Name).To(Equal(""))
			Expect(destinations[0].Description).To(Equal(""))
			Expect(destinations[0].Protocol).To(Equal("tcp"))
			Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
			Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
			Expect(destinations[0].ICMPType).To(Equal(-1))
			Expect(destinations[0].ICMPCode).To(Equal(-1))

			By("GetByGUID")
			destination, err := egressDestinationTable.GetByGUID(tx, terminalId)
			Expect(err).NotTo(HaveOccurred())

			_, err = uuid.ParseHex(destination.GUID)
			Expect(err).NotTo(HaveOccurred())

			Expect(destination.GUID).To(Equal(terminalId))
			Expect(destination.Name).To(Equal(""))
			Expect(destination.Description).To(Equal(""))
			Expect(destination.Protocol).To(Equal("tcp"))
			Expect(destination.IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
			Expect(destination.Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
			Expect(destination.ICMPType).To(Equal(-1))
			Expect(destination.ICMPCode).To(Equal(-1))

			_, err = egressDestinationTable.GetByGUID(tx, "garbage id")
			Expect(err).NotTo(HaveOccurred())

			By("Delete")
			err = egressDestinationTable.Delete(tx, "garbage id")
			// TODO: ensure that there's zero count deleted on the response, and that the list is empty
			// right now the response contains an empty destination object and a total count == 1
			Expect(err).ToNot(HaveOccurred())

			err = egressDestinationTable.Delete(tx, terminalId)
			Expect(err).NotTo(HaveOccurred())

			destinations, err = egressDestinationTable.All(tx)
			Expect(err).NotTo(HaveOccurred())
			Expect(destinations).To(HaveLen(0))
		})
	})

	Context("when a destination metadata exist for destination", func() {
		BeforeEach(func() {
			metadataTable := store.DestinationMetadataTable{}
			_, err = metadataTable.Create(tx, terminalId, "dest name", "dest desc")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("GetByGUID", func() {
			It("returns the name/description", func() {
				By("All")

				destinations, err := egressDestinationTable.All(tx)
				Expect(err).NotTo(HaveOccurred())

				_, err = uuid.ParseHex(destinations[0].GUID)
				Expect(err).NotTo(HaveOccurred())

				Expect(destinations[0].GUID).To(Equal(terminalId))
				Expect(destinations[0].Name).To(Equal("dest name"))
				Expect(destinations[0].Description).To(Equal("dest desc"))
				Expect(destinations[0].Protocol).To(Equal("tcp"))
				Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
				Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
				Expect(destinations[0].ICMPType).To(Equal(-1))
				Expect(destinations[0].ICMPCode).To(Equal(-1))

				By("GetByGUID")

				destination, err := egressDestinationTable.GetByGUID(tx, terminalId)
				Expect(err).NotTo(HaveOccurred())

				_, err = uuid.ParseHex(destination.GUID)
				Expect(err).NotTo(HaveOccurred())

				Expect(destination.GUID).To(Equal(terminalId))
				Expect(destination.Name).To(Equal("dest name"))
				Expect(destination.Description).To(Equal("dest desc"))
				Expect(destination.Protocol).To(Equal("tcp"))
				Expect(destination.IPRanges).To(Equal([]store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}}))
				Expect(destination.Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))
				Expect(destination.ICMPType).To(Equal(-1))
				Expect(destination.ICMPCode).To(Equal(-1))
			})
		})
	})
})
