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
	"github.com/tedsuo/rata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Get", func() {
	var dataStore *fakes.Store
	var marshaler *fakes.Marshaler
	var container models.Container
	var logger *fakes.Logger
	var handler http.Handler
	var request *http.Request

	BeforeEach(func() {
		dataStore = &fakes.Store{}
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		logger = &fakes.Logger{}
		getHandler := &handlers.GetHandler{
			Store:     dataStore,
			Marshaler: marshaler,
			Logger:    logger,
		}
		container = models.Container{ID: "some-container"}
		dataStore.GetReturns(container, nil)

		handler, request = rataWrap(getHandler, "GET", "/containers/:container_id", rata.Params{"container_id": "some-container"})
	})

	It("should return a requested container as JSON", func() {
		req, err := http.NewRequest("GET", "/containers/some-container", nil)
		Expect(err).NotTo(HaveOccurred())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		var receivedContainer models.Container
		err = json.Unmarshal(resp.Body.Bytes(), &receivedContainer)
		Expect(err).NotTo(HaveOccurred())
		Expect(receivedContainer).To(Equal(container))

		Expect(dataStore.GetArgsForCall(0)).To(Equal("some-container"))
	})

	It("should marshal the container received from the datastore", func() {
		req, err := http.NewRequest("GET", "/containers/some-container", nil)
		Expect(err).NotTo(HaveOccurred())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		Expect(marshaler.MarshalCallCount()).To(Equal(1))
		Expect(marshaler.MarshalArgsForCall(0)).To(Equal(container))
	})

	Context("when there are no containers", func() {
		BeforeEach(func() {
			dataStore.GetReturns(models.Container{}, store.NotFoundError)
		})

		It("should return a 404", func() {
			req, err := http.NewRequest("GET", "/containers/some-container", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("when an error occurs on container get", func() {
		var theError error = errors.New("WUT")
		BeforeEach(func() {
			dataStore.GetReturns(models.Container{}, theError)
		})

		It("should return a 500 and log the error", func() {
			req, err := http.NewRequest("GET", "/containers/some-container", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))

			Expect(logger.ErrorCallCount()).To(Equal(1))
			action, err, _ := logger.ErrorArgsForCall(0)
			Expect(action).To(Equal("store-get"))
			Expect(err).To(Equal(theError))
		})
	})

	Context("when marshaling fails", func() {
		It("should return a 500 error", func() {
			marshaler.MarshalReturns(nil, errors.New("teapot"))
			req, err := http.NewRequest("GET", "/containers/some-container", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("when writing the response body fails", func() {
		It("should log the error", func() {
			req, err := http.NewRequest("GET", "/containers/some-container", nil)
			Expect(err).NotTo(HaveOccurred())

			badResponseWriter := &badResponseWriter{}
			handler.ServeHTTP(badResponseWriter, req)

			Expect(logger.ErrorCallCount()).To(Equal(1))
			action, err, _ := logger.ErrorArgsForCall(0)
			Expect(action).To(Equal("container-get"))
			Expect(err).To(MatchError("failed writing body: some bad writer"))
		})
	})
})
