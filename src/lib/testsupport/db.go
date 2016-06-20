package testsupport

import (
	"fmt"
	"lib/db"
	"os"
	"os/exec"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type DBConnectionInfo struct {
	Type     string
	Hostname string
	Port     string
	Username string
	Password string
}

type TestDatabase struct {
	Name     string
	ConnInfo *DBConnectionInfo
}

func (d *TestDatabase) DBConfig() db.Config {
	port, err := strconv.Atoi(d.ConnInfo.Port)
	Expect(err).NotTo(HaveOccurred())

	return db.Config{
		Type:     d.ConnInfo.Type,
		Host:     d.ConnInfo.Hostname,
		Port:     port,
		Username: d.ConnInfo.Username,
		Password: d.ConnInfo.Password,
		Name:     d.Name,
		SSLMode:  "disable",
	}
}

func (d *TestDatabase) Destroy() {
	d.ConnInfo.RemoveDatabase(d)
}

func (c *DBConnectionInfo) CreateDatabase(dbName string) *TestDatabase {
	testDB := &TestDatabase{Name: dbName, ConnInfo: c}
	_, err := c.execSQL(fmt.Sprintf("CREATE DATABASE %s", dbName))
	Expect(err).NotTo(HaveOccurred())
	return testDB
}

func (c *DBConnectionInfo) RemoveDatabase(db *TestDatabase) {
	_, err := c.execSQL(fmt.Sprintf("DROP DATABASE %s", db.Name))
	Expect(err).NotTo(HaveOccurred())
}

func (c *DBConnectionInfo) execSQL(sqlCommand string) (string, error) {
	var cmd *exec.Cmd

	if c.Type == "mysql" {
		cmd = exec.Command("mysql",
			"-h", c.Hostname,
			"-P", c.Port,
			"-u", c.Username,
			"-e", sqlCommand)
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+c.Password)
	} else if c.Type == "postgres" {
		cmd = exec.Command("psql",
			"-h", c.Hostname,
			"-p", c.Port,
			"-U", c.Username,
			"-c", sqlCommand)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+c.Password)
	} else {
		panic("unsupported database type: " + c.Type)
	}

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "9s").Should(gexec.Exit())
	if session.ExitCode() != 0 {
		return "", fmt.Errorf("unexpected exit code: %d", session.ExitCode())
	}
	return string(session.Out.Contents()), nil
}

func GetPostgresDBConnectionInfo() *DBConnectionInfo {
	return &DBConnectionInfo{
		Type:     "postgres",
		Hostname: "127.0.0.1",
		Port:     "5432",
		Username: "postgres",
		Password: "",
	}
}

func GetMySQLDBConnectionInfo() *DBConnectionInfo {
	return &DBConnectionInfo{
		Type:     "mysql",
		Hostname: "127.0.0.1",
		Port:     "3306",
		Username: "root",
		Password: "password",
	}
}

func GetDBConnectionInfo() *DBConnectionInfo {
	if os.Getenv("MYSQL") == "true" {
		return GetMySQLDBConnectionInfo()
	} else {
		return GetPostgresDBConnectionInfo()
	}
}
