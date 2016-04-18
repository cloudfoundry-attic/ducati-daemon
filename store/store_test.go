package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"

	"github.com/cloudfoundry-incubator/ducati-daemon/db"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/cloudfoundry-incubator/ducati-daemon/testsupport"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var dataStore store.Store
	var testDatabase *testsupport.TestDatabase
	var realDb *sqlx.DB
	var mockDb *fakes.Db

	BeforeEach(func() {
		mockDb = &fakes.Db{}

		dbName := fmt.Sprintf("test_database_%x", rand.Int())
		dbConnectionInfo := testsupport.GetDBConnectionInfo()
		testDatabase = dbConnectionInfo.CreateDatabase(dbName)
		var err error
		realDb, err = db.GetConnectionPool(testDatabase.URL())
		Expect(err).NotTo(HaveOccurred())
		dataStore, err = store.New(realDb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		if testDatabase != nil {
			testDatabase.Destroy()
		}
	})

	Describe("Connecting to the database and migrating", func() {
		Context("when the tables already exist", func() {
			It("succeeds", func() {
				_, err := store.New(realDb)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.ExecReturns(nil, errors.New("some error"))
			})

			It("should return a sensible error", func() {
				_, err := store.New(mockDb)
				Expect(err).To(MatchError("setting up tables: some error"))
			})
		})
	})

	Describe("round-tripping a container through the database", func() {
		It("stores and retrieves all the fields on the container model", func() {
			toCreate := models.Container{
				NetworkID:   "some-crazy-network-id",
				ID:          "some-container-id",
				MAC:         "01:02:03:04:05:06",
				IP:          "192.168.100.2",
				HostIP:      "10.11.12.13",
				SandboxName: "vni-99",
				App:         "some-app-guid",
			}

			Expect(dataStore.Create(toCreate)).To(Succeed())

			retrieved, err := dataStore.Get("some-container-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(retrieved).To(Equal(toCreate))
		})
	})

	Describe("Create", func() {
		It("saves the container", func() {
			container := models.Container{ID: "some-id"}

			err := dataStore.Create(container)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when a container with the same id already exists", func() {
			It("should return a RecordExistsError", func() {
				container := models.Container{ID: "some-id"}

				err := dataStore.Create(container)
				Expect(err).NotTo(HaveOccurred())

				containerDuplicate := models.Container{ID: "some-id"}

				err = dataStore.Create(containerDuplicate)
				Expect(err).To(Equal(store.RecordExistsError))
			})
		})

		Context("when the db operation fails", func() {
			Context("when the failure is an unexpected pq error", func() {

				BeforeEach(func() {
					mockDb.NamedExecReturns(nil,
						&pq.Error{
							Code: "2201G",
						})
				})

				It("should return the error code", func() {
					store, err := store.New(mockDb)
					Expect(err).NotTo(HaveOccurred())

					err = store.Create(models.Container{})
					Expect(err).To(MatchError("insert: invalid_argument_for_width_bucket_function"))
				})
			})

			Context("when the failure is not a pq Error", func() {

				BeforeEach(func() {
					mockDb.NamedExecReturns(nil, errors.New("some-insert-error"))
				})

				It("should return a sensible error", func() {
					store, err := store.New(mockDb)
					Expect(err).NotTo(HaveOccurred())

					err = store.Create(models.Container{})
					Expect(err).To(MatchError("insert: some-insert-error"))
				})
			})
		})
	})

	Describe("Get", func() {
		Context("when the container has been put in the store", func() {
			var expectedContainer models.Container

			BeforeEach(func() {
				expectedContainer = models.Container{
					ID: "some-container",
				}

				err := dataStore.Create(expectedContainer)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can retrieve the container", func() {
				container, err := dataStore.Get(expectedContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(container).To(Equal(expectedContainer))
			})
		})

		Context("when the container has not been put in the store", func() {
			It("should return a RecordNotFoundError", func() {
				_, err := dataStore.Get("some-unknown-id")
				Expect(err).To(Equal(store.RecordNotFoundError))
			})
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.GetReturns(errors.New("some get error"))
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get("doesnt-matter")
				Expect(err).To(MatchError("getting record: some get error"))
			})
		})
	})

	Describe("All", func() {
		var expectedContainers []models.Container

		BeforeEach(func() {
			expectedContainers = []models.Container{
				{ID: "some-id-1", NetworkID: "some-network-id-1"},
				{ID: "some-id-2", NetworkID: "some-network-id-2"},
				{ID: "some-id-3", NetworkID: "some-network-id-2"},
			}

			for _, c := range expectedContainers {
				Expect(dataStore.Create(c)).To(Succeed())
			}
		})

		It("returns all containers that have been added", func() {
			containers, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(containers).To(ConsistOf(expectedContainers))
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.SelectReturns(errors.New("some select error"))
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.All()
				Expect(err).To(MatchError("listing all: some select error"))
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			theContainers := []models.Container{
				{ID: "some-id-1"},
				{ID: "some-id-2"},
				{ID: "some-id-3"},
			}

			for _, c := range theContainers {
				Expect(dataStore.Create(c)).To(Succeed())
			}
		})

		Context("when there is a container to delete", func() {
			It("should remove the container", func() {
				Expect(dataStore.Delete("some-id-2")).To(Succeed())
				Expect(dataStore.All()).To(ConsistOf(
					[]models.Container{
						{ID: "some-id-1"},
						{ID: "some-id-3"},
					}))
			})
		})

		Context("when there is no container with the given id", func() {
			It("should return a RecordNotFoundError", func() {
				Expect(dataStore.Delete("doesn't-exist")).To(Equal(store.RecordNotFoundError))
			})
		})

		Context("when the db operation fails", func() {
			BeforeEach(func() {
				mockDb.ExecStub = func(string, ...interface{}) (sql.Result, error) {
					if mockDb.ExecCallCount() == 2 {
						return nil, errors.New("some delete error")
					}
					return nil, nil
				}
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb)
				Expect(err).NotTo(HaveOccurred())

				err = store.Delete("doesnt-matter")
				Expect(err).To(MatchError("deleting: some delete error"))
			})
		})

		Context("when looking for the RowsAffected() returns an error", func() {
			BeforeEach(func() {
				mockExecResult := &fakes.SqlResult{}
				mockExecResult.RowsAffectedReturns(0, errors.New("some rows affected error"))

				mockDb.ExecStub = func(string, ...interface{}) (sql.Result, error) {
					if mockDb.ExecCallCount() == 2 {
						return mockExecResult, nil
					}
					return nil, nil
				}
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb)
				Expect(err).NotTo(HaveOccurred())

				err = store.Delete("doesnt-matter")
				Expect(err).To(MatchError("deleting: rows affected: some rows affected error"))
			})
		})

		Context("when the number of rows affected is not 1", func() {
			BeforeEach(func() {
				mockExecResult := &fakes.SqlResult{}
				mockExecResult.RowsAffectedReturns(-1, nil)

				mockDb.ExecStub = func(string, ...interface{}) (sql.Result, error) {
					if mockDb.ExecCallCount() == 2 {
						return mockExecResult, nil
					}
					return nil, nil
				}
			})

			It("should return a sensible error", func() {
				store, err := store.New(mockDb)
				Expect(err).NotTo(HaveOccurred())

				err = store.Delete("doesnt-matter")
				Expect(err).To(MatchError("deleting: rows affected: -1"))
			})
		})
	})
})
