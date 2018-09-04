package store_test

import (
	"fmt"
	"policy-server/db"
	"policy-server/store"
	testhelpers "test-helpers"
	"time"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressDestination", func() {
	var (
		dbConf dbHelper.Config
		realDb *db.ConnWrapper

		egressPolicyTable      *store.EgressPolicyTable
		egressDestinationTable *store.EgressDestinationTable
	)

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("egress_destination_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Egress Destination Test")
		realDb = db.NewConnectionPool(dbConf, 200, 200, "Egress Destination Test", "Egress Destination Test", logger)

		migrate(realDb)

		egressDestinationTable = &store.EgressDestinationTable{}

		tx, err := realDb.Beginx()
		Expect(err).NotTo(HaveOccurred())

		terminalId, err := egressPolicyTable.CreateTerminal(tx)
		Expect(err).NotTo(HaveOccurred())

		_, err = egressDestinationTable.CreateIPRange(tx, terminalId, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, -1, -1)
		Expect(err).NotTo(HaveOccurred())

		err = tx.Commit()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testhelpers.RemoveDatabase(dbConf)
	})

	Context("when a destination metadata doesn't exist for destination", func() {
		It("returns empty strings for name/description", func() {
			tx, err := realDb.Beginx()
			Expect(err).NotTo(HaveOccurred())

			destinations, err := egressDestinationTable.All(tx)
			Expect(err).NotTo(HaveOccurred())

			err = tx.Rollback()
			Expect(err).NotTo(HaveOccurred())

			Expect(destinations).To(Equal([]store.EgressDestination{
				{
					ID:          "1",
					Name:        "",
					Description: "",
					Protocol:    "tcp",
					IPRanges:    []store.IPRange{{Start: "1.1.1.1", End: "2.2.2.2"}},
					Ports:       []store.Ports{{Start: 8080, End: 8081}},
					ICMPType:    -1,
					ICMPCode:    -1,
				},
			}))
		})
	})
})
