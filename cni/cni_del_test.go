package cni_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/cni"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CniDel", func() {
	var (
		datastore     *fakes.Store
		deletor       *fakes.Deletor
		controller    *cni.DelController
		ipAllocator   *fakes.IPAllocator
		networkMapper *fakes.NetworkMapper
		payload       models.CNIDelPayload
	)

	BeforeEach(func() {
		datastore = &fakes.Store{}
		deletor = &fakes.Deletor{}
		ipAllocator = &fakes.IPAllocator{}
		networkMapper = &fakes.NetworkMapper{}

		networkMapper.GetVNIReturns(42, nil)
		datastore.GetReturns(models.Container{
			NetworkID: "some-network-id",
		}, nil)

		controller = &cni.DelController{
			Datastore:     datastore,
			Deletor:       deletor,
			IPAllocator:   ipAllocator,
			NetworkMapper: networkMapper,
		}

		payload = models.CNIDelPayload{
			InterfaceName:      "some-interface-name",
			ContainerNamespace: "/some/container/namespace/path",
			ContainerID:        "some-container-id",
		}
	})

	It("gets the network id from the datastore", func() {
		err := controller.Del(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(datastore.GetCallCount()).To(Equal(1))
		containerID := datastore.GetArgsForCall(0)
		Expect(containerID).To(Equal("some-container-id"))
	})

	Context("when getting the record from the datastore fails", func() {
		BeforeEach(func() {
			datastore.GetReturns(models.Container{}, errors.New("some error"))
		})

		It("aborts and returns a wrapped error", func() {
			err := controller.Del(payload)
			Expect(err).To(MatchError("datastore get: some error"))

			Expect(networkMapper.GetVNICallCount()).To(Equal(0))
		})
	})

	It("uses the network id to get the VNI", func() {
		err := controller.Del(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(networkMapper.GetVNICallCount()).To(Equal(1))
		Expect(networkMapper.GetVNIArgsForCall(0)).To(Equal("some-network-id"))
	})

	Context("when getting the VNI fails", func() {
		BeforeEach(func() {
			networkMapper.GetVNIReturns(0, errors.New("some error"))
		})

		It("aborts and returns a wrapped error", func() {
			err := controller.Del(payload)
			Expect(err).To(MatchError("get vni: some error"))

			Expect(deletor.DeleteCallCount()).To(Equal(0))
		})
	})

	It("deletes the container from the network", func() {
		err := controller.Del(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(deletor.DeleteCallCount()).To(Equal(1))

		ifName, cnsPath, sbName, vxName := deletor.DeleteArgsForCall(0)
		Expect(ifName).To(Equal("some-interface-name"))
		Expect(cnsPath).To(Equal("/some/container/namespace/path"))
		Expect(sbName).To(Equal("vni-42"))
		Expect(vxName).To(Equal("vxlan42"))
	})

	Context("when deleting the container from the network fails", func() {
		BeforeEach(func() {
			deletor.DeleteReturns(errors.New("some-deletor-error"))
		})

		It("aborts and returns a wrapped error", func() {
			err := controller.Del(payload)

			Expect(err).To(MatchError("deletor: some-deletor-error"))
			Expect(datastore.DeleteCallCount()).To(Equal(0))
		})
	})

	It("deletes the container from the datastore", func() {
		err := controller.Del(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(datastore.DeleteCallCount()).To(Equal(1))
		containerID := datastore.DeleteArgsForCall(0)
		Expect(containerID).To(Equal("some-container-id"))
	})

	Context("when deleting from the datastore fails", func() {
		BeforeEach(func() {
			datastore.DeleteReturns(errors.New("some-datastore-error"))
		})

		It("returns a wrapped error", func() {
			err := controller.Del(payload)
			Expect(err).To(MatchError("datastore delete: some-datastore-error"))
		})
	})

	It("releases the IP allocation", func() {
		err := controller.Del(payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(ipAllocator.ReleaseIPCallCount()).To(Equal(1))

		networkID, containerID := ipAllocator.ReleaseIPArgsForCall(0)
		Expect(networkID).To(Equal("some-network-id"))
		Expect(containerID).To(Equal("some-container-id"))
	})

	Context("when releasing the IP fails", func() {
		BeforeEach(func() {
			ipAllocator.ReleaseIPReturns(errors.New("mango"))
		})

		It("returns a wrapped error", func() {
			err := controller.Del(payload)
			Expect(err).To(MatchError("release ip: mango"))
		})
	})
})
