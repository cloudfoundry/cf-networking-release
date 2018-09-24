package main

import (
	"fmt"
	"os"
	"policy-server/config"
	"policy-server/store"
	"time"

	"flag"
	"lib/common"
	"log"
	"policy-server/store/migrations"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
)

const (
	jobPrefix = "policy-server-migrate-db"
	logPrefix = "cfnetworking"
)

func main() {
	err := mainWithError()
	if err != nil {
		fmt.Printf("fatal error occured, %s", err)
		os.Exit(1)
	}
}

func mainWithError() error {
	conf := parseConfig()

	logger, _ := lagerflags.NewFromConfig(fmt.Sprintf("%s.%s", logPrefix, jobPrefix), common.GetLagerConfig())

	doneChan := make(chan bool, 1)
	go func() {
		for {
			err := migrateAndPopulateGroupsTable(logger, conf)
			if err != nil {
				logger.Error("failed migrating and populating tags, retrying", err)
				time.Sleep(1 * time.Second)
				continue
			}
			doneChan <- true
			return
		}
	}()

	select {
	case <-doneChan:
		return nil
	case <-time.After(time.Duration(conf.DatabaseMigrationTimeout) * time.Second):
		return fmt.Errorf("migrations and groups table population timed out after %d seconds", conf.DatabaseMigrationTimeout)
	}
}

func parseConfig() *config.Config {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("%s.%s: could not read config file: %s", logPrefix, jobPrefix, err)
	}

	return conf
}

func migrateAndPopulateGroupsTable(logger lager.Logger, conf *config.Config) error {
	logger.Info("getting migration db connection")
	dbConn, err := db.NewConnectionPool(
		conf.Database,
		conf.MaxOpenConnections,
		conf.MaxIdleConnections,
		time.Duration(conf.MaxConnectionsLifetimeSeconds)*time.Second,
		logPrefix,
		jobPrefix,
		logger,
	)
	if err != nil {
		return fmt.Errorf("getting migration db connection: %s", err)
	}

	defer dbConn.Close()

	logger.Info("migration db connection retrieved")

	migrator := &migrations.Migrator{
		MigrateAdapter: &migrations.MigrateAdapter{},
		MigrationsProvider: &migrations.MigrationsProvider{
			Store: &store.MigrationsStore{
				DBConn: dbConn,
			},
		},
	}

	tagPopulator := &store.TagPopulator{DBConnection: dbConn}

	logger.Info("running migrations")
	numMigrationsRun, err := migrator.PerformMigrations(dbConn.DriverName(), dbConn, 0)
	if err != nil {
		return fmt.Errorf("perform migrations: %s", err)
	}
	logger.Info("finished running migrations", lager.Data{"num-migrations-completed": numMigrationsRun})

	logger.Info("populating groups table")
	err = tagPopulator.PopulateTables(conf.TagLength)
	if err != nil {
		return fmt.Errorf("populating groups table: %s", err)
	}
	logger.Info("finished populating groups table")

	return nil
}
