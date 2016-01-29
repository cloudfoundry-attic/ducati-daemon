package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Post", func() {
	var dataStore *fakes.Store
	var handler *handlers.PostHandler
	var request *http.Request
	var container models.Container
	var unmarshaler *fakes.Unmarshaler
	var logger *fakes.Logger

	BeforeEach(func() {
		dataStore = &fakes.Store{}
		logger = &fakes.Logger{}

		unmarshaler = &fakes.Unmarshaler{}
		unmarshaler.UnmarshalStub = json.Unmarshal

		handler = &handlers.PostHandler{
			Store:       dataStore,
			Unmarshaler: unmarshaler,
			Logger:      logger,
		}

		container = models.Container{
			ID: "my-new-container",
		}

		dataStore.CreateReturns(nil)

		var err error
		request, err = http.NewRequest("POST", "/containers", strings.NewReader(`{ "id": "my-new-container" }`))
		Expect(err).NotTo(HaveOccurred())
	})

	It("adds a container to the store", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(unmarshaler.UnmarshalCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusCreated))

		Expect(dataStore.CreateCallCount()).To(Equal(1))
		Expect(dataStore.CreateArgsForCall(0)).To(Equal(container))
	})

	Context("when unmarshaling fails", func() {
		It("should return a 400 error", func() {
			unmarshaler.UnmarshalReturns(errors.New("boom"))
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))

			Expect(dataStore.CreateCallCount()).To(Equal(0))
		})
	})

	Context("when the store Create fails", func() {
		Context("because the record already exists", func() {
			It("should return 409 conflict", func() {
				resp := httptest.NewRecorder()
				request, err := http.NewRequest("POST", "/containers", strings.NewReader(`{ "id": "my-new-container" }`))
				Expect(err).NotTo(HaveOccurred())

				dataStore.CreateReturns(store.RecordExistsError)
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusConflict))
			})
		})

		It("should return a 502 and log the error", func() {
			request, err := http.NewRequest("POST", "/containers", strings.NewReader(`{"ID": "something"}`))
			Expect(err).NotTo(HaveOccurred())

			resp := httptest.NewRecorder()

			dataStore.CreateReturns(errors.New("go away"))
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadGateway))

			Expect(logger.ErrorCallCount()).To(Equal(1))
			action, err, _ := logger.ErrorArgsForCall(0)
			Expect(action).To(Equal("store-put"))
			Expect(err).To(MatchError("go away"))
		})
	})

	Context("when the request body reader fails", func() {
		It("should log the error and not attempt to respond", func() {
			aBadReader := &badReader{}
			request, err := http.NewRequest("POST", "/containers", aBadReader)
			Expect(err).NotTo(HaveOccurred())

			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))

			Expect(dataStore.CreateCallCount()).To(Equal(0))
			Expect(unmarshaler.UnmarshalCallCount()).To(Equal(0))
		})
	})
})

type badReader struct{}

func (r *badReader) Read(buffer []byte) (int, error) {
	return 0, errors.New("bad")
}
