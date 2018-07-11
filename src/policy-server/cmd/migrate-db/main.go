package main

import (
	"bytes"
	"code.cloudfoundry.org/lager"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"policy-server/cmd/common"
	"policy-server/config"
	"policy-server/db"
	"policy-server/store/migrations"
)

const (
	jobPrefix    = "policy-server-migrate-db"
	logPrefix    = "cfnetworking"
	MaxTagLength = 3
	MinTagLength = 1
)

func main() {
	err := mainWithError()
	if err != nil {
		fmt.Printf("fatal error occured, %s", err)
		os.Exit(1)
	}
}

func mainWithError() error {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("%s.%s: could not read config file: %s", logPrefix, jobPrefix, err)
		return err
	}

	//move this validation to the config struct or template rendering
	if conf.TagLength < MinTagLength || conf.TagLength > MaxTagLength {
		return fmt.Errorf("tag length out of range (%d-%d): %d",
			MinTagLength,
			MaxTagLength,
			conf.TagLength,
		)
	}

	logger := lager.NewLogger(fmt.Sprintf("%s.%s", logPrefix, jobPrefix))
	logger.RegisterSink(common.InitLoggerSink(logger, "DEBUG"))

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

	migrator := &migrations.Migrator{
		MigrateAdapter: &migrations.MigrateAdapter{},
	}

	logger.Info("running migrations", lager.Data{})
	numMigrationsRun, err := migrator.PerformMigrations(dbConn.DriverName(), dbConn, 0)
	if err != nil {
		return fmt.Errorf("perform migrations: %s", err)
	}

	logger.Info("finished running migrations", lager.Data{
		"num-migrations-completed": numMigrationsRun,
	})

	logger.Info("populating groups table", lager.Data{})
	err = populateGroupsTable(dbConn, conf.TagLength)
	logger.Info("finished populating groups table", lager.Data{})

	return err
}

func populateGroupsTable(dbConn *db.ConnWrapper, tagLength int) error {
	var err error
	row := dbConn.QueryRow(`SELECT COUNT(*) FROM groups`)
	if row != nil {
		var count int
		err = row.Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}

	var b bytes.Buffer
	_, err = b.WriteString("INSERT INTO groups (guid) VALUES (NULL)")
	if err != nil {
		return err
	}

	for i := 1; i < int(math.Exp2(float64(tagLength*8)))-1; i++ {
		_, err = b.WriteString(", (NULL)")
		if err != nil {
			return err
		}
	}

	_, err = dbConn.Exec(b.String())
	if err != nil {
		return err
	}

	return nil
}
