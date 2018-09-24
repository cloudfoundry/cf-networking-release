package testhelpers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"policy-server/db"

	configHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/lager"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func CreateDatabase(config configHelper.Config) {
	config.Timeout = 120
	dbToCreate := config.DatabaseName
	config.DatabaseName = ""
	fmt.Fprintf(ginkgo.GinkgoWriter, "%s Creating database %s", time.Now().String(), dbToCreate)
	logger := lager.NewLogger("Test Support")
	connectionPool := db.NewConnectionPool(
		config,
		200,
		200,
		5*time.Minute,
		"testsupport",
		"db-helper",
		logger,
	)
	defer connectionPool.Close()
	_, err := connectionPool.Exec(fmt.Sprintf("CREATE DATABASE %s", dbToCreate))
	Expect(err).NotTo(HaveOccurred())
}

func RemoveDatabase(config configHelper.Config) {
	config.Timeout = 120

	dbToDrop := config.DatabaseName
	config.DatabaseName = ""

	logger := lager.NewLogger("Test Support")
	connectionPool := db.NewConnectionPool(
		config,
		200,
		200,
		5*time.Minute,
		"testsupport",
		"db-helper",
		logger,
	)
	defer connectionPool.Close()
	_, err := connectionPool.Exec(fmt.Sprintf("DROP DATABASE %s", dbToDrop))
	if err != nil {
		fmt.Fprintln(ginkgo.GinkgoWriter, fmt.Sprintf("%+v", err))
	}
}

const DefaultDBTimeout = 5

func getPostgresDBConfig() configHelper.Config {
	return configHelper.Config{
		Type:     "postgres",
		User:     "postgres",
		Password: "",
		Host:     "127.0.0.1",
		Port:     5432,
		Timeout:  DefaultDBTimeout,
	}
}

func getMySQLDBConfig() configHelper.Config {
	return configHelper.Config{
		Type:     "mysql",
		User:     "root",
		Password: "password",
		Host:     "127.0.0.1",
		Port:     3306,
		Timeout:  DefaultDBTimeout,
	}
}

func GetDBConfig() configHelper.Config {
	dbEnv := os.Getenv("DB")
	switch {
	case strings.HasPrefix(dbEnv, "mysql"):
		return getMySQLDBConfig()
	case strings.HasPrefix(dbEnv, "postgres"):
		return getPostgresDBConfig()
	default:
		panic("unable to determine database to use.  Set environment variable DB")
	}
}
