package store_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var dataStore store.Store

	BeforeEach(func() {
		dataStore = store.New()
	})

	Describe("Put", func() {
		It("saves the container", func() {
			container := models.Container{ID: "some-id"}

			err := dataStore.Put(container)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Get", func() {
		Context("when the container has been put in the store", func() {
			var expectedContainer models.Container

			BeforeEach(func() {
				expectedContainer = models.Container{
					ID: "some-container",
				}

				err := dataStore.Put(expectedContainer)
				Expect(err).NotTo(HaveOccurred())
			})

			It("can retrieve the container", func() {
				container, err := dataStore.Get(expectedContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(container).To(Equal(expectedContainer))
			})
		})

		Context("when the container has not been put in the store", func() {
			It("raises an error", func() {
				_, err := dataStore.Get("some-unknown-id")
				Expect(err).To(MatchError("container not found: some-unknown-id"))
			})
		})
	})

	Describe("All", func() {
		var expectedContainers []models.Container

		BeforeEach(func() {
			expectedContainers = []models.Container{
				models.Container{ID: "some-id-1"},
				models.Container{ID: "some-id-2"},
				models.Container{ID: "some-id-3"},
			}

			for _, c := range expectedContainers {
				Expect(dataStore.Put(c)).To(Succeed())
			}
		})

		It("returns all containers that have been added", func() {
			containers, err := dataStore.All()
			Expect(err).NotTo(HaveOccurred())
			Expect(containers).To(ConsistOf(expectedContainers))
		})
	})
})
