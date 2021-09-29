package integration_test

import (
	"fmt"
	"math"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/ports"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/integration/helpers"
	"code.cloudfoundry.org/policy-server/store/migrations"
	testhelpers "code.cloudfoundry.org/test-helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const TimeoutShort = 20 * time.Second

var _ = Describe("Migrate DB Binary", func() {
	var (
		dbConf db.Config
		conf   config.Config
	)

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("migrate_test_node_%d", ports.PickAPort())

		conf, _ = helpers.DefaultTestConfig(dbConf, "unused", "fixtures")
		conf.Database = dbConf
	})

	Context("when the db is available", func() {
		BeforeEach(func() {
			testhelpers.CreateDatabase(dbConf)
		})

		AfterEach(func() {
			testhelpers.RemoveDatabase(dbConf)
		})

		It("runs the migrations and seeds the groups table", func() {
			session := helpers.RunMigrationsPreStartBinary(migrateDbPath, conf)
			Eventually(session.Wait(TimeoutShort)).Should(gexec.Exit(0))

			conn := createDbConn(dbConf)
			defer conn.Close()

			assertMigrationsSucceeded(conn, conf)
		})

		Context("when the migrations have already run", func() {
			It("runs successfully", func() {
				session := helpers.RunMigrationsPreStartBinary(migrateDbPath, conf)
				Eventually(session.Wait(TimeoutShort)).Should(gexec.Exit(0))
				session = helpers.RunMigrationsPreStartBinary(migrateDbPath, conf)
				Eventually(session.Wait(TimeoutShort)).Should(gexec.Exit(0))
			})
		})
	})

	Context("when the db is not available", func() {
		Context("when it becomes available", func() {
			AfterEach(func() {
				testhelpers.RemoveDatabase(dbConf)
			})

			It("eventually succeeds", func() {
				session := helpers.RunMigrationsPreStartBinary(migrateDbPath, conf)
				testhelpers.CreateDatabase(dbConf)
				Eventually(session.Wait(TimeoutShort)).Should(gexec.Exit(0))
				conn := createDbConn(dbConf)
				defer conn.Close()

				assertMigrationsSucceeded(conn, conf)
			})
		})

		Context("when it never becomes available", func() {
			It("exits non-zero", func() {
				conf.DatabaseMigrationTimeout = 1
				session := helpers.RunMigrationsPreStartBinary(migrateDbPath, conf)
				Eventually(session.Wait(TimeoutShort)).Should(gexec.Exit(1))
			})
		})
	})
})

func assertMigrationsSucceeded(conn *db.ConnWrapper, conf config.Config) {
	numMigrations := len(migrations.V1ModifiedMigrationsToPerform) +
		len(migrations.V2ModifiedMigrationsToPerform) +
		len(migrations.V3ModifiedMigrationsToPerform) +
		len(migrations.MigrationsToPerform)

	var migrationCount int
	conn.QueryRow("SELECT COUNT(*) FROM gorp_migrations").Scan(&migrationCount)
	Expect(migrationCount).To(Equal(numMigrations))

	var groupCount int
	conn.QueryRow(`SELECT COUNT(*) FROM "groups"`).Scan(&groupCount)
	Expect(groupCount).To(Equal(int(math.Exp2(float64(conf.TagLength*8))) - 1))
}

func createDbConn(dbConf db.Config) *db.ConnWrapper {
	conn, err := db.NewConnectionPool(
		dbConf,
		1,
		1,
		5*time.Minute,
		"test-db",
		"test-job-prefix",
		lagertest.NewTestLogger("test"),
	)
	Expect(err).NotTo(HaveOccurred())
	return conn
}
