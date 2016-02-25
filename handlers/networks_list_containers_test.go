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
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"
)

var _ = Describe("GET /networks/:network_id", func() {
	var (
		getHandler *handlers.NetworksListContainers
		marshaler  *fakes.Marshaler
		logger     *lagertest.TestLogger
		datastore  *fakes.Store
	)

	BeforeEach(func() {
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		datastore = &fakes.Store{}
		logger = lagertest.NewTestLogger("test")
		getHandler = &handlers.NetworksListContainers{
			Marshaler: marshaler,
			Logger:    logger,
			Datastore: datastore,
		}

		datastore.AllReturns([]models.Container{
			{ID: "container-id-1", IP: "192.168.0.1", NetworkID: "network-id-1"},
			{ID: "container-id-2", IP: "192.168.0.2", NetworkID: "network-id-1"},
			{ID: "container-id-199", IP: "192.168.0.3", NetworkID: "network-id-2"},
			{ID: "container-id-200", IP: "192.168.0.4", NetworkID: "network-id-2"},
		}, nil)
	})

	It("should return a matching set of containers as json", func() {
		handler, request := rataWrap(getHandler, "GET", "/networks/:network_id", rata.Params{"network_id": "network-id-1"})
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		var receivedContainers []models.Container
		err := json.Unmarshal(resp.Body.Bytes(), &receivedContainers)
		Expect(err).NotTo(HaveOccurred())

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(receivedContainers).To(Equal([]models.Container{
			{ID: "container-id-1", IP: "192.168.0.1", NetworkID: "network-id-1"},
			{ID: "container-id-2", IP: "192.168.0.2", NetworkID: "network-id-1"},
		}))
	})

	Context("when the datastore fails", func() {
		BeforeEach(func() {
			datastore.AllReturns(nil, errors.New("nothing for you"))
		})

		It("responds with a 500", func() {
			handler, request := rataWrap(getHandler, "GET", "/networks/:network_id", rata.Params{"network_id": "network-id-1"})
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say("networks-list-containers.*datastore-all-failed.*nothing for you"))
		})
	})

	Context("when marshaling fails", func() {
		BeforeEach(func() {
			marshaler.MarshalReturns(nil, errors.New("bang"))
		})

		It("responds with a 500", func() {
			handler, request := rataWrap(getHandler, "GET", "/networks/:network_id", rata.Params{"network_id": "network-id-1"})
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say("networks-list-containers.*marshal-failed.*bang"))
		})
	})
})
