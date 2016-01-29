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
			It("should return a NotFoundError", func() {
				_, err := dataStore.Get("some-unknown-id")
				Expect(err).To(Equal(store.NotFoundError))
			})
		})
	})

	Describe("All", func() {
		var expectedContainers []models.Container

		BeforeEach(func() {
			expectedContainers = []models.Container{
				{ID: "some-id-1"},
				{ID: "some-id-2"},
				{ID: "some-id-3"},
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

	Describe("Delete", func() {
		BeforeEach(func() {
			theContainers := []models.Container{
				{ID: "some-id-1"},
				{ID: "some-id-2"},
				{ID: "some-id-3"},
			}

			for _, c := range theContainers {
				Expect(dataStore.Put(c)).To(Succeed())
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
			It("should return a NotFoundError", func() {
				Expect(dataStore.Delete("doesn't-exist")).To(Equal(store.NotFoundError))
			})
		})
	})
})
