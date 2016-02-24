package acceptance_test

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/cloudfoundry-incubator/ducati-daemon/testsupport"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var ducatidPath string
var dbConnInfo *testsupport.DBConnectionInfo

func TestDucatid(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ducati Daemon Acceptance Suite")
}

type beforeSuiteData struct {
	DucatidPath string
	DBConnInfo  testsupport.DBConnectionInfo
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// only run on node 1
	ducatidPath, err := gexec.Build("github.com/cloudfoundry-incubator/ducati-daemon/cmd/ducatid")
	Expect(err).NotTo(HaveOccurred())

	dbConnInfo := testsupport.GetDBConnectionInfo()

	bytesToMarshal, err := json.Marshal(beforeSuiteData{
		DucatidPath: ducatidPath,
		DBConnInfo:  *dbConnInfo,
	})
	Expect(err).NotTo(HaveOccurred())

	return bytesToMarshal
}, func(marshaledBytes []byte) {
	// run on all nodes
	var data beforeSuiteData
	Expect(json.Unmarshal(marshaledBytes, &data)).To(Succeed())
	ducatidPath = data.DucatidPath
	dbConnInfo = &data.DBConnInfo

	rand.Seed(config.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = SynchronizedAfterSuite(func() {
	// run on all nodes
}, func() {
	// run only on node 1
	gexec.CleanupBuildArtifacts()

})

var testDatabase *testsupport.TestDatabase

var _ = BeforeEach(func() {
	dbName := fmt.Sprintf("test_db_%x", rand.Int31())
	testDatabase = dbConnInfo.CreateDatabase(dbName)
})

var _ = AfterEach(func() {
	dbConnInfo.RemoveDatabase(testDatabase)
})
