package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ListContainers", func() {
	var dataStore *fakes.Store
	var handler *handlers.ListContainers
	var marshaler *fakes.Marshaler
	var containers []models.Container
	var logger *lagertest.TestLogger

	BeforeEach(func() {
		dataStore = &fakes.Store{}
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		logger = lagertest.NewTestLogger("test")
		handler = &handlers.ListContainers{
			Marshaler: marshaler,
			Logger:    logger,
			Datastore: dataStore,
		}

		containers = []models.Container{
			{ID: "container-id-1", IP: "192.168.0.1", NetworkID: "network-id-1"},
			{ID: "container-id-2", IP: "192.168.0.2", NetworkID: "network-id-1"},
			{ID: "container-id-199", IP: "192.168.0.3", NetworkID: "network-id-2"},
			{ID: "container-id-200", IP: "192.168.0.4", NetworkID: "network-id-2"},
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

		Expect(resp.Code).To(Equal(http.StatusOK))
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

	Context("when listing from the store fails", func() {
		It("should return a 500 error and log", func() {
			dataStore.AllReturns(nil, errors.New("teapot"))

			req, err := http.NewRequest("GET", "/containers", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say("list-containers.*datastore-all-failed.*teapot"))
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
			Expect(logger).To(gbytes.Say("list-containers.*marshal-failed.*teapot"))
		})
	})
})
