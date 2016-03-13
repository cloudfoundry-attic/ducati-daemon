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
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/testsupport"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"
)

var _ = Describe("NetworksSetupContainer", func() {
	var (
		unmarshaler         *fakes.Unmarshaler
		logger              *lagertest.TestLogger
		datastore           *fakes.Store
		ipamResult          *types.Result
		creator             *fakes.Creator
		handler             http.Handler
		request             *http.Request
		osLocker            *fakes.OSThreadLocker
		marshaler           *fakes.Marshaler
		ipAllocator         *fakes.IPAllocator
		networkMapper       *fakes.NetworkMapper
		expectedResultBytes []byte
	)

	BeforeEach(func() {
		osLocker = &fakes.OSThreadLocker{}

		unmarshaler = &fakes.Unmarshaler{}
		unmarshaler.UnmarshalStub = json.Unmarshal

		logger = lagertest.NewTestLogger("test")
		datastore = &fakes.Store{}
		creator = &fakes.Creator{}

		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		ipAllocator = &fakes.IPAllocator{}
		networkMapper = &fakes.NetworkMapper{}

		setupHandler := &handlers.NetworksSetupContainer{
			Unmarshaler:    unmarshaler,
			Logger:         logger,
			Datastore:      datastore,
			Creator:        creator,
			OSThreadLocker: osLocker,
			Marshaler:      marshaler,
			IPAllocator:    ipAllocator,
			NetworkMapper:  networkMapper,
		}

		ipamResult = &types.Result{
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

		var err error
		expectedResultBytes, err = json.Marshal(ipamResult)
		Expect(err).NotTo(HaveOccurred())

		ipAllocator.AllocateIPReturns(ipamResult, nil)

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
		request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(`{}`)))
	})

	It("sets up the container network", func() {
		payload, err := json.Marshal(models.NetworksSetupContainerPayload{
			Args:               "FOO=BAR;ABC=123",
			ContainerNamespace: "/some/namespace/path",
			InterfaceName:      "interface-name",
		})
		Expect(err).NotTo(HaveOccurred())
		networkMapper.GetVNIReturns(99, nil)

		request.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusCreated))
		Expect(creator.SetupCallCount()).To(Equal(1))
		Expect(creator.SetupArgsForCall(0)).To(Equal(container.CreatorConfig{
			NetworkID:       "network-id-1",
			ContainerNsPath: "/some/namespace/path",
			ContainerID:     "container-id",
			InterfaceName:   "interface-name",
			IPAMResult:      ipamResult,
			VNI:             99,
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

	It("locks and unlocks the os thread", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(osLocker.LockOSThreadCallCount()).To(Equal(1))
		Expect(osLocker.UnlockOSThreadCallCount()).To(Equal(1))
	})

	It("uses the network id to get the VNI", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(networkMapper.GetVNICallCount()).To(Equal(1))
		Expect(networkMapper.GetVNIArgsForCall(0)).To(Equal("network-id-1"))
	})

	Context("when getting the VNI fails", func() {
		BeforeEach(func() {
			networkMapper.GetVNIReturns(0, errors.New("some error"))
		})

		It("logs the error and responds with status code 500", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say("network-mapper-get-vni.*some error"))
		})

		It("does not attempt to allocate an IP or call create", func() {
			Expect(ipAllocator.AllocateIPCallCount()).To(Equal(0))
			Expect(creator.SetupCallCount()).To(Equal(0))
		})
	})

	Context("when there are errors", func() {
		Context("when the request body cannot be read", func() {
			BeforeEach(func() {
				request.Body = ioutil.NopCloser(&testsupport.BadReader{})
			})

			It("should log and respond with status 400", func() {
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(logger).To(gbytes.Say("networks-setup-containers.*body-read-failed"))
			})
		})

		Context("when unmarshalling payload fails", func() {
			BeforeEach(func() {
				request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(`{}`)))
			})

			It("logs an error and responds with code 400", func() {
				unmarshaler.UnmarshalReturns(errors.New("some-unmarshal-error"))
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusBadRequest))
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

	Describe("IP allocation", func() {
		It("allocates an IP and returns the json ipamResult", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(ipAllocator.AllocateIPCallCount()).To(Equal(1))
			Expect(resp.Body.String()).To(MatchJSON(expectedResultBytes))

			Expect(marshaler.MarshalCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusCreated))

			networkID, containerID := ipAllocator.AllocateIPArgsForCall(0)
			Expect(networkID).To(Equal("network-id-1"))
			Expect(containerID).To(Equal("container-id"))
		})

		Context("when things go wrong", func() {
			Context("when the allocator returns a NoMoreAddressesError", func() {
				It("should log and return a 409 status with JSON body encoding the error message", func() {
					ipAllocator.AllocateIPReturns(nil, ipam.NoMoreAddressesError)
					resp := httptest.NewRecorder()
					handler.ServeHTTP(resp, request)

					Expect(resp.Body.String()).To(MatchJSON(`{ "error": "no addresses available" }`))
					Expect(logger).To(gbytes.Say(`networks-setup-containers.allocate-ip.*no addresses available`))
					Expect(resp.Code).To(Equal(http.StatusConflict))
				})

				Context("when marshaling the error fails", func() {
					BeforeEach(func() {
						ipAllocator.AllocateIPReturns(nil, ipam.NoMoreAddressesError)
						marshaler.MarshalReturns([]byte(`bad`), errors.New("banana"))
					})

					It("should log the error", func() {
						resp := httptest.NewRecorder()
						handler.ServeHTTP(resp, request)

						Expect(resp.Body.String()).To(BeEmpty())
						Expect(logger).To(gbytes.Say("allocate-ip-error-marshaling.*banana"))
						Expect(resp.Code).To(Equal(http.StatusConflict))
					})
				})
			})

			Context("when the allocator errors in some other fashion", func() {
				It("should return 500 and log the error", func() {
					ipAllocator.AllocateIPReturns(nil, errors.New("tomato"))
					resp := httptest.NewRecorder()
					handler.ServeHTTP(resp, request)

					Expect(logger).To(gbytes.Say("networks-setup-containers.allocate-ip.*tomato"))
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				})
			})

		})

		Context("when marshaling the result fails", func() {
			It("should return 500 and log the error", func() {
				marshaler.MarshalReturns([]byte(`bad`), errors.New("banana"))

				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(logger).To(gbytes.Say("networks-setup-containers.allocate-ip.*banana"))
				Expect(resp.Body.String()).To(BeEmpty())
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when writing the response body fails", func() {
			It("should log the error", func() {
				badResponseWriter := &badResponseWriter{}
				handler.ServeHTTP(badResponseWriter, request)

				Expect(logger).To(gbytes.Say("networks-setup-containers.allocate-ip.*failed writing body: some bad writer"))
			})
		})
	})
})
