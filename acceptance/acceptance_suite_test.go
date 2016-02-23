package acceptance_test

import (
	"fmt"
	"math/rand"
	"os/exec"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var ducatidPath string
var postgresSession *gexec.Session

func TestDucatid(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ducati Daemon Acceptance Suite")
}

func waitForPostgresToBoot() error {
	cmd := exec.Command("psql",
		"-h", "localhost",
		"-p", "5432",
		"-U", "postgres",
		"-c", `\conninfo`)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
	if session.ExitCode() != 0 {
		return fmt.Errorf("unexpected exit code: %d", session.ExitCode())
	}
	Expect(session.Out).To(gbytes.Say(`You are connected to database "postgres"`))
	return nil
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// only run on node 1
	ducatidPath, err := gexec.Build("github.com/cloudfoundry-incubator/ducati-daemon/cmd/ducatid")
	Expect(err).NotTo(HaveOccurred())

	cmd := exec.Command("/docker-entrypoint.sh", "postgres")
	postgresSession, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(postgresSession, "5s").Should(gbytes.Say("PostgreSQL init process complete; ready for start up"))

	Eventually(waitForPostgresToBoot, "5s").Should(Succeed())
	return []byte(ducatidPath)
}, func(pathsByte []byte) {
	// run on all nodes
	ducatidPath = string(pathsByte)

	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {
	// run on all nodes
}, func() {
	// run only on node 1
	postgresSession.Interrupt()
	Eventually(postgresSession, "5s").Should(gexec.Exit(0))

})

var dbName string
var databaseURL string

func createDatabase(dbName string) string {
	cmd := exec.Command("psql",
		"-h", "localhost",
		"-p", "5432",
		"-U", "postgres",
		"-c", fmt.Sprintf("CREATE DATABASE %s", dbName))
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		"postgres", "", "localhost", dbName, "disable")
}

func removeDatabase(dbName string) {
	cmd := exec.Command("psql",
		"-h", "localhost",
		"-p", "5432",
		"-U", "postgres",
		"-c", fmt.Sprintf("DROP DATABASE %s", dbName))
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
}

var _ = BeforeEach(func() {
	dbName = fmt.Sprintf("test_db_%x", rand.Int31())
	databaseURL = createDatabase(dbName)
})

var _ = AfterEach(func() {
	removeDatabase(dbName)
})
