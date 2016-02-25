package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	exec_fakes "github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"
)

var _ = Describe("NetworksSetupContainer", func() {
	var (
		setupHandler *handlers.NetworksSetupContainer
		unmarshaler  *fakes.Unmarshaler
		logger       *lagertest.TestLogger
		datastore    *fakes.Store
		executor     *exec_fakes.Executor
		ipamResult   types.Result
		creator      *fakes.Creator
		handler      http.Handler
		request      *http.Request
	)

	BeforeEach(func() {
		unmarshaler = &fakes.Unmarshaler{}
		unmarshaler.UnmarshalStub = json.Unmarshal

		logger = lagertest.NewTestLogger("test")
		datastore = &fakes.Store{}
		executor = &exec_fakes.Executor{}
		creator = &fakes.Creator{}

		setupHandler = &handlers.NetworksSetupContainer{
			Unmarshaler: unmarshaler,
			Logger:      logger,
			Datastore:   datastore,
			Creator:     creator,
		}

		ipamResult = types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("192.168.100.2"),
					Mask: net.CIDRMask(24, 32),
				},
				Gateway: net.ParseIP("192.168.100.1"),
				Routes: []types.Route{{
					Dst: net.IPNet{
						IP:   net.ParseIP("192.168.1.5"),
						Mask: net.CIDRMask(24, 32),
					},
					GW: net.ParseIP("192.168.1.1"),
				}, {
					Dst: net.IPNet{
						IP:   net.ParseIP("192.168.2.5"),
						Mask: net.CIDRMask(24, 32),
					},
					GW: net.ParseIP("192.168.1.99"),
				}},
			},
		}

		creator.SetupReturns(models.Container{
			ID:        "container-id",
			NetworkID: "network-id-1",
			MAC:       "00:00:00:00:00",
			HostIP:    "10.12.100.4",
			IP:        "192.168.160.3",
		}, nil)

		handler, request = rataWrap(setupHandler, "POST", "/networks/:network_id/:container_id", rata.Params{
			"network_id":   "network-id-1",
			"container_id": "container-id",
		})
	})

	It("sets up the container network", func() {
		payload, err := json.Marshal(models.NetworksSetupContainerPayload{
			Args:               "FOO=BAR;ABC=123",
			ContainerNamespace: "/some/namespace/path",
			InterfaceName:      "interface-name",
			VNI:                99,
			HostIP:             "10.12.100.4",
			IPAM:               ipamResult,
		})
		Expect(err).NotTo(HaveOccurred())

		request.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusCreated))
		Expect(creator.SetupCallCount()).To(Equal(1))
		Expect(creator.SetupArgsForCall(0)).To(Equal(container.CreatorConfig{
			NetworkID:       "network-id-1",
			BridgeName:      "vxlanbr99",
			ContainerNsPath: "/some/namespace/path",
			ContainerID:     "container-id",
			InterfaceName:   "interface-name",
			VNI:             99,
			HostIP:          "10.12.100.4",
			IPAMResult:      ipamResult,
		}))

		Expect(datastore.CreateCallCount()).To(Equal(1))
		Expect(datastore.CreateArgsForCall(0)).To(Equal(models.Container{
			ID:        "container-id",
			NetworkID: "network-id-1",
			MAC:       "00:00:00:00:00",
			HostIP:    "10.12.100.4",
			IP:        "192.168.160.3",
		}))
	})

	Context("when there are errors", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(`{}`)))
		})

		Context("when unmarshalling payload fails", func() {
			It("logs an error and 500s", func() {
				unmarshaler.UnmarshalReturns(errors.New("some-unmarshal-error"))
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(logger).To(gbytes.Say("networks-setup-containers.unmarshal-failed.*some-unmarshal-error"))

				Expect(creator.SetupCallCount()).To(BeZero())
			})
		})

		Context("when container creation fails", func() {
			It("logs an error and 500s", func() {
				creator.SetupReturns(models.Container{}, errors.New("some-container-setup-error"))
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(logger).To(gbytes.Say("networks-setup-containers.container-setup-failed.*some-container-setup-error"))

				Expect(datastore.CreateCallCount()).To(BeZero())
			})
		})

		Context("when datastore create fails", func() {
			It("logs an error and 500s", func() {
				datastore.CreateReturns(errors.New("some-datastore-create-error"))
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(logger).To(gbytes.Say("networks-setup-containers.datastore-create-failed.*some-datastore-create-error"))
			})
		})
	})
})
