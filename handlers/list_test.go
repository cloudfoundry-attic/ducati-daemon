package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List", func() {
	var dataStore store.Store
	var handler *handlers.ListHandler

	BeforeEach(func() {
		dataStore = store.New()
		handler = &handlers.ListHandler{
			Store: dataStore,
		}
	})

	It("should return the containers as a JSON list", func() {
		containers := []models.Container{
			models.Container{ID: "some-container"},
			models.Container{ID: "some-other-container"},
		}

		for _, c := range containers {
			err := dataStore.Put(c)
			Expect(err).NotTo(HaveOccurred())
		}

		req, err := http.NewRequest("GET", "/containers", nil)
		Expect(err).NotTo(HaveOccurred())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		var receivedContainers []models.Container
		err = json.Unmarshal(resp.Body.Bytes(), &receivedContainers)
		Expect(err).NotTo(HaveOccurred())
		Expect(receivedContainers).To(ConsistOf(containers))
	})

	Context("when there are no containers", func() {
		It("should return an empty list", func() {
			req, err := http.NewRequest("GET", "/containers", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Body.String()).To(MatchJSON(`[]`))
		})
	})
})
