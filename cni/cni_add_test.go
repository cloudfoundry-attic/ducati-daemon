package cni_test

import (
	"errors"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/cni"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CniAdd", func() {
	var (
		datastore     *fakes.Store
		ipamResult    *types.Result
		creator       *fakes.Creator
		controller    *cni.AddController
		osLocker      *fakes.OSThreadLocker
		ipAllocator   *fakes.IPAllocator
		networkMapper *fakes.NetworkMapper
		payload       models.CNIAddPayload
	)

	BeforeEach(func() {
		osLocker = &fakes.OSThreadLocker{}

		datastore = &fakes.Store{}
		creator = &fakes.Creator{}

		ipAllocator = &fakes.IPAllocator{}
		networkMapper = &fakes.NetworkMapper{}

		controller = &cni.AddController{
			Datastore:      datastore,
			Creator:        creator,
			OSThreadLocker: osLocker,
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
		Expect(err).NotTo(HaveOccurred())

		ipAllocator.AllocateIPReturns(ipamResult, nil)

		creator.SetupReturns(models.Container{
			ID:        "container-id",
			NetworkID: "network-id-1",
			App:       "app-id-1",
			MAC:       "00:00:00:00:00",
			HostIP:    "10.12.100.4",
			IP:        "192.168.160.3",
		}, nil)

		payload = models.CNIAddPayload{
			Args:               "FOO=BAR;ABC=123",
			ContainerNamespace: "/some/namespace/path",
			InterfaceName:      "interface-name",
			Network: models.NetworkPayload{
				ID:  "network-id-1",
				App: "app-id-1",
			},
			ContainerID: "container-id",
		}
	})

	It("sets up the container network", func() {
		networkMapper.GetVNIReturns(99, nil)

		_, err := controller.Add(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(creator.SetupCallCount()).To(Equal(1))
		Expect(creator.SetupArgsForCall(0)).To(Equal(container.CreatorConfig{
			NetworkID:       "network-id-1",
			App:             "app-id-1",
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
			App:       "app-id-1",
			MAC:       "00:00:00:00:00",
			HostIP:    "10.12.100.4",
			IP:        "192.168.160.3",
		}))
	})

	It("locks and unlocks the os thread", func() {
		_, err := controller.Add(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(osLocker.LockOSThreadCallCount()).To(Equal(1))
		Expect(osLocker.UnlockOSThreadCallCount()).To(Equal(1))
	})

	It("uses the network id to get the VNI", func() {
		_, err := controller.Add(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(networkMapper.GetVNICallCount()).To(Equal(1))
		Expect(networkMapper.GetVNIArgsForCall(0)).To(Equal("network-id-1"))
	})

	It("allocates an IP and returns the ipamResult", func() {
		returnedIPAMResult, err := controller.Add(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(ipAllocator.AllocateIPCallCount()).To(Equal(1))
		networkID, containerID := ipAllocator.AllocateIPArgsForCall(0)
		Expect(networkID).To(Equal("network-id-1"))
		Expect(containerID).To(Equal("container-id"))

		Expect(returnedIPAMResult).To(BeIdenticalTo(ipamResult))
	})

	Context("when getting the VNI fails", func() {
		It("aborts and returns a wrapped error", func() {
			networkMapper.GetVNIReturns(0, errors.New("some error"))

			_, err := controller.Add(payload)
			Expect(err).To(MatchError("get vni: some error"))

			Expect(ipAllocator.AllocateIPCallCount()).To(Equal(0))
		})
	})

	Context("when the allocator returns an error", func() {
		It("aborts and returns the error without wrapping it", func() {
			allocateError := errors.New("some, possibly typed error")
			ipAllocator.AllocateIPReturns(nil, allocateError)
			_, err := controller.Add(payload)

			Expect(err).To(BeIdenticalTo(allocateError))
		})
	})

	Context("when container creation fails", func() {
		It("aborts and returns a wrapped error", func() {
			creator.SetupReturns(models.Container{}, errors.New("some error"))
			_, err := controller.Add(payload)

			Expect(datastore.CreateCallCount()).To(BeZero())
			Expect(err).To(MatchError("container setup: some error"))
		})
	})

	Context("when datastore create fails", func() {
		It("returns a wrapped error", func() {
			datastore.CreateReturns(errors.New("some error"))
			_, err := controller.Add(payload)
			Expect(err).To(MatchError("datastore create: some error"))
		})
	})
})
