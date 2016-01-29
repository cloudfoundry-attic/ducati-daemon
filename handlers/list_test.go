package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List", func() {
	var dataStore *fakes.Store
	var handler *handlers.ListHandler
	var marshaler *fakes.Marshaler
	var containers []models.Container

	BeforeEach(func() {
		dataStore = &fakes.Store{}
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		handler = &handlers.ListHandler{
			Store:     dataStore,
			Marshaler: marshaler,
		}
		containers = []models.Container{
			models.Container{ID: "some-container"},
			models.Container{ID: "some-other-container"},
		}
		dataStore.AllReturns(containers, nil)
	})

	It("should return the containers as a JSON list", func() {
		req, err := http.NewRequest("GET", "/containers", nil)
		Expect(err).NotTo(HaveOccurred())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		var receivedContainers []models.Container
		err = json.Unmarshal(resp.Body.Bytes(), &receivedContainers)
		Expect(err).NotTo(HaveOccurred())
		Expect(receivedContainers).To(ConsistOf(containers))
	})

	It("should marshal the containers received from the datastore", func() {
		req, err := http.NewRequest("GET", "/containers", nil)
		Expect(err).NotTo(HaveOccurred())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		Expect(marshaler.MarshalCallCount()).To(Equal(1))
		Expect(marshaler.MarshalArgsForCall(0)).To(Equal(containers))
	})

	Context("when there are no containers", func() {
		BeforeEach(func() {
			dataStore.AllReturns([]models.Container{}, nil)
		})

		It("should return an empty list", func() {
			req, err := http.NewRequest("GET", "/containers", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Body.String()).To(MatchJSON(`[]`))
		})
	})

	Context("when marshaling fails", func() {
		It("should return a 500 error", func() {
			marshaler.MarshalReturns(nil, errors.New("teapot"))
			req, err := http.NewRequest("GET", "/containers", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
