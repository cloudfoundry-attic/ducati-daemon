package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"
)

var _ = Describe("GET /container/:container_id", func() {
	var (
		getHandler *handlers.GetContainer
		marshaler  *fakes.Marshaler
		logger     *lagertest.TestLogger
		datastore  *fakes.Store
	)

	BeforeEach(func() {
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		datastore = &fakes.Store{}
		logger = lagertest.NewTestLogger("test")
		getHandler = &handlers.GetContainer{
			Marshaler: marshaler,
			Logger:    logger,
			Datastore: datastore,
		}

		datastore.GetStub = func(id string) (models.Container, error) {
			return models.Container{
				ID:        "container-id-1",
				IP:        "192.168.0.1",
				NetworkID: "network-id-1",
			}, nil
		}
	})

	It("should return, as json, the info for a container with given id", func() {
		handler, request := rataWrap(getHandler, "GET", "/containers/:container_id", rata.Params{"container_id": "container-id-1"})
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		var receivedContainer models.Container
		err := json.Unmarshal(resp.Body.Bytes(), &receivedContainer)
		Expect(err).NotTo(HaveOccurred())

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(receivedContainer).To(Equal(models.Container{
			ID:        "container-id-1",
			IP:        "192.168.0.1",
			NetworkID: "network-id-1",
		}))

	})

	Context("when the record does not exist in the database", func() {
		BeforeEach(func() {
			datastore.GetReturns(models.Container{}, store.RecordNotFoundError)
		})
		It("logs the error and returns with a 404", func() {
			handler, request := rataWrap(getHandler, "GET", "/containers/:container_id", rata.Params{"container_id": "some-nonexistent-container-id"})
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("get-container.*record-not-found"))
			Expect(resp.Code).To(Equal(http.StatusNotFound))
			Expect(resp.Body.String()).To(BeEmpty())
		})
	})

	Context("when the database in an unknown way", func() {
		BeforeEach(func() {
			datastore.GetReturns(models.Container{}, errors.New("some-database-error"))
		})
		It("logs the error and returns with a 500", func() {
			handler, request := rataWrap(getHandler, "GET", "/containers/:container_id", rata.Params{"container_id": "some-container-id"})
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("get-container.*database-error"))
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(BeEmpty())
		})
	})

	Context("when the marshaling fails", func() {
		BeforeEach(func() {
			marshaler.MarshalReturns([]byte("some-junk"), errors.New("bang"))
		})

		It("logs the error and returns with a 500", func() {
			handler, request := rataWrap(getHandler, "GET", "/containers/:container_id", rata.Params{"container_id": "container-id-1"})
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("get-container.*marshal-failed.*bang"))
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(BeEmpty())
		})
	})
})
