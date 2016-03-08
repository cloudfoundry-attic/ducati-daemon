package db_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/db"
	"github.com/cloudfoundry-incubator/ducati-daemon/testsupport"
	"github.com/nu7hatch/gouuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pg", func() {
	var (
		testDatabase *testsupport.TestDatabase
		dbName       string
	)

	BeforeEach(func() {
		guid, err := uuid.NewV4()
		Expect(err).NotTo(HaveOccurred())

		dbName = fmt.Sprintf("test_%x", guid[:])
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)
	})

	AfterEach(func() {
		if testDatabase != nil {
			testDatabase.Destroy()
			testDatabase = nil
		}
	})

	Describe("GetConnectionPool", func() {
		It("returns a database reference", func() {
			database, err := db.GetConnectionPool(testDatabase.URL())
			Expect(err).NotTo(HaveOccurred())
			defer database.Close()

			var databaseName string
			err = database.QueryRow("SELECT current_database();").Scan(&databaseName)
			Expect(err).NotTo(HaveOccurred())
			Expect(databaseName).To(Equal(dbName))
		})

		Context("when the database cannot be accessed", func() {
			It("returns an error", func() {
				url := testDatabase.URL()

				testDatabase.Destroy()
				testDatabase = nil

				_, err := db.GetConnectionPool(url)
				Expect(err).To(MatchError(ContainSubstring("unable to ping")))
			})
		})
	})
})
