package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var dataStore *fakes.Store
	var handler *handlers.DeleteHandler
	var marshaler *fakes.Marshaler
	var logger *fakes.Logger

	BeforeEach(func() {
		dataStore = &fakes.Store{}
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		logger = &fakes.Logger{}
		handler = &handlers.DeleteHandler{
			Store:  dataStore,
			Logger: logger,
		}
		dataStore.DeleteReturns(nil)
	})

	It("should return a 204 when container is deleted", func() {
		req, err := http.NewRequest("DELETE", "/containers/some-container", nil)
		Expect(err).NotTo(HaveOccurred())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		Expect(resp.Code).To(Equal(http.StatusNoContent))
	})

	Context("when the container did not exist", func() {
		BeforeEach(func() {
			dataStore.DeleteReturns(store.NotFoundError)
		})

		It("should return a 404", func() {
			req, err := http.NewRequest("DELETE", "/containers/some-container", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("when an error occurs on container delete", func() {
		BeforeEach(func() {
			dataStore.DeleteReturns(errors.New("WUT"))
		})

		It("should return a 500 and log the error", func() {
			req, err := http.NewRequest("DELETE", "/containers/some-container", nil)
			Expect(err).NotTo(HaveOccurred())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))

			Expect(logger.ErrorCallCount()).To(Equal(1))
			action, err, _ := logger.ErrorArgsForCall(0)
			Expect(action).To(Equal("store-delete"))
			Expect(err).To(MatchError("WUT"))
		})
	})
})
