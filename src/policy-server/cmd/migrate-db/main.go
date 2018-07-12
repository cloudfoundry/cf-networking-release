package main

import (
	"code.cloudfoundry.org/lager"
	"flag"
	"fmt"
	"os"
	"policy-server/cmd/common"
	"policy-server/config"
	"policy-server/db"
	"policy-server/store"
	"policy-server/store/migrations"
	"log"
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
	logger := logger()
	dbConn := dbConnection(conf, logger)

	err := migrateDb(dbConn, logger)
	if err != nil {
		return fmt.Errorf("perform migrations: %s", err)
	}

	return populateGroupsTable(dbConn, conf.TagLength, logger)
}

func logger() lager.Logger {
	logger := lager.NewLogger(fmt.Sprintf("%s.%s", logPrefix, jobPrefix))
	logger.RegisterSink(common.InitLoggerSink(logger, "DEBUG"))
	return logger
}

func dbConnection(conf *config.Config, logger lager.Logger) *db.ConnWrapper {
	logger.Info("getting migration db connection", lager.Data{})
	dbConn := db.NewConnectionPool(
		conf.Database,
		conf.MaxOpenConnections,
		conf.MaxIdleConnections,
		logPrefix,
		jobPrefix,
		logger,
	)
	logger.Info("migration db connection retrieved", lager.Data{})
	return dbConn
}

func migrateDb(dbConn *db.ConnWrapper, logger lager.Logger) error {
	logger.Info("running migrations", lager.Data{})
	migrator := &migrations.Migrator{MigrateAdapter: &migrations.MigrateAdapter{}}
	numMigrationsRun, err := migrator.PerformMigrations(dbConn.DriverName(), dbConn, 0)
	if err != nil {
		return err
	}

	logger.Info("finished running migrations", lager.Data{
		"num-migrations-completed": numMigrationsRun,
	})
	return nil
}

func populateGroupsTable(dbConn *db.ConnWrapper, tagLength int, logger lager.Logger) error {
	logger.Info("populating groups table", lager.Data{})

	tagPopulator := &store.TagPopulator{DBConnection: dbConn}
	err := tagPopulator.PopulateTables(tagLength)

	logger.Info("finished populating groups table", lager.Data{})
	return err
}

func parseConfig() (*config.Config) {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("%s.%s: could not read config file: %s", logPrefix, jobPrefix, err)
	}

	return conf
}
