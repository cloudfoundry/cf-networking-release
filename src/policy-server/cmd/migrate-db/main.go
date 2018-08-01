package main

import (
	"code.cloudfoundry.org/lager"
	"flag"
	"fmt"
	"log"
	"os"
	"policy-server/cmd/common"
	"policy-server/config"
	"policy-server/db"
	"policy-server/store"
	"policy-server/store/migrations"
	"time"
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
	c := make(chan error, 1)
	go func() {
		err := migrateAndPopulateGroupsTable(conf)
		c <- err
	}()

	timeoutDuration := time.Duration(conf.DatabaseMigrationTimeout) * time.Second
	select {
	case err := <-c:
		return err
	case <-time.After(timeoutDuration):
		return fmt.Errorf("migrations and groups table population timed out after %d seconds", conf.DatabaseMigrationTimeout)
	}
}

func migrateAndPopulateGroupsTable(conf *config.Config) error {

	logger := logger()
	dbConn := dbConnection(conf, logger)

	err := migrateDb(dbConn, logger)
	if err != nil {
		return fmt.Errorf("perform migrations: %s, and close db error: %s", err, dbConn.Close())
	}

	err = populateGroupsTable(dbConn, conf.TagLength, logger)
	if err != nil {
		return fmt.Errorf("populating groups table: %s, and close db error: %s", err, dbConn.Close())
	}
	
	return dbConn.Close()
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
	migrator := &migrations.Migrator{
		MigrateAdapter: &migrations.MigrateAdapter{},
		MigrationsProvider: &migrations.MigrationsProvider{
			Store: &store.MigrationsStore{
				DBConn: dbConn,
			},
		},
	}
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

func parseConfig() *config.Config {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("%s.%s: could not read config file: %s", logPrefix, jobPrefix, err)
	}

	return conf
}
