package testsupport

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/cloudfoundry-incubator/ducati-daemon/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type DBConnectionInfo struct {
	Hostname string
	Port     string
	Username string
	Password string
}

type TestDatabase struct {
	name     string
	connInfo *DBConnectionInfo
}

func (d *TestDatabase) URL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.connInfo.Username, d.connInfo.Password, d.connInfo.Hostname, d.connInfo.Port, d.name, "disable")
}

func (d *TestDatabase) Destroy() {
	d.connInfo.RemoveDatabase(d)
}

func (d *TestDatabase) AsDaemonConfig() config.Database {
	port, err := strconv.Atoi(d.connInfo.Port)
	if err != nil {
		panic(err)
	}
	return config.Database{
		Host:     d.connInfo.Hostname,
		Port:     port,
		Username: d.connInfo.Username,
		Password: d.connInfo.Password,
		Name:     d.name,
		SslMode:  "disable",
	}
}

func (c *DBConnectionInfo) CreateDatabase(dbName string) *TestDatabase {
	testDB := &TestDatabase{name: dbName, connInfo: c}
	_, err := c.execSQL(fmt.Sprintf("CREATE DATABASE %s", dbName))
	Expect(err).NotTo(HaveOccurred())
	return testDB
}

func (c *DBConnectionInfo) RemoveDatabase(db *TestDatabase) {
	_, err := c.execSQL(fmt.Sprintf("DROP DATABASE %s", db.name))
	Expect(err).NotTo(HaveOccurred())
}

func (c *DBConnectionInfo) execSQL(sqlCommand string) (string, error) {
	cmd := exec.Command("psql",
		"-h", c.Hostname,
		"-p", c.Port,
		"-U", c.Username,
		"-c", sqlCommand)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+c.Password)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "9s").Should(gexec.Exit())
	if session.ExitCode() != 0 {
		return "", fmt.Errorf("unexpected exit code: %d", session.ExitCode())
	}
	return string(session.Out.Contents()), nil
}

func GetDBConnectionInfo() *DBConnectionInfo {
	return &DBConnectionInfo{
		Hostname: "localhost",
		Port:     "5432",
		Username: "postgres",
		Password: "",
	}
}
