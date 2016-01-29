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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Post", func() {
	var dataStore *fakes.Store
	var handler *handlers.PostHandler
	var request *http.Request
	var container models.Container
	var unmarshaler *fakes.Unmarshaler

	BeforeEach(func() {
		dataStore = &fakes.Store{}

		unmarshaler = &fakes.Unmarshaler{}
		unmarshaler.UnmarshalStub = json.Unmarshal

		handler = &handlers.PostHandler{
			Store:       dataStore,
			Unmarshaler: unmarshaler,
		}

		container = models.Container{
			ID: "my-new-container",
		}

		dataStore.PutReturns(nil)

		var err error
		request, err = http.NewRequest("POST", "/containers", strings.NewReader(`{ "id": "my-new-container" }`))
		Expect(err).NotTo(HaveOccurred())
	})

	It("adds a container to the store", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(unmarshaler.UnmarshalCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusNoContent))

		Expect(dataStore.PutCallCount()).To(Equal(1))
		Expect(dataStore.PutArgsForCall(0)).To(Equal(container))
	})

	Context("when unmarshaling fails", func() {
		It("should return a 400 error", func() {
			unmarshaler.UnmarshalReturns(errors.New("boom"))
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))

			Expect(dataStore.PutCallCount()).To(Equal(0))
		})
	})

	Context("when the store Put fails", func() {
		It("should return a 502 error", func() {
			request, err := http.NewRequest("POST", "/containers", strings.NewReader(`{"ID": "something"}`))
			Expect(err).NotTo(HaveOccurred())

			resp := httptest.NewRecorder()

			dataStore.PutReturns(errors.New("go away"))
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusBadGateway))
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

			Expect(dataStore.PutCallCount()).To(Equal(0))
			Expect(unmarshaler.UnmarshalCallCount()).To(Equal(0))
		})
	})
})

type badReader struct{}

func (r *badReader) Read(buffer []byte) (int, error) {
	return 0, errors.New("bad")
}
