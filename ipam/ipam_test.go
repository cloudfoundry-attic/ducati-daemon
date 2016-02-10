package ipam_test

import (
	"errors"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ipam", func() {
	var (
		store         *fakes.AllocatorStore
		storeFactory  *fakes.StoreFactory
		storeLocker   *fakes.Locker
		configFactory *fakes.ConfigFactory
		configLocker  *fakes.Locker
		allocator     *ipam.Allocator
	)

	BeforeEach(func() {
		store = &fakes.AllocatorStore{}
		storeFactory = &fakes.StoreFactory{}
		storeLocker = &fakes.Locker{}

		configFactory = &fakes.ConfigFactory{}
		configLocker = &fakes.Locker{}

		allocator = ipam.New(storeFactory, storeLocker, configFactory, configLocker)
	})

	Describe("AllocateIP", func() {
		var allocated map[string]struct{}
		var config ipam.Config

		BeforeEach(func() {
			config = ipam.Config{
				Subnet: net.IPNet{
					IP:   net.ParseIP("192.168.2.0"),
					Mask: net.CIDRMask(24, 32),
				},
			}
			configFactory.CreateReturns(config, nil)

			allocated = map[string]struct{}{}
			store.ReserveStub = func(containerID string, ip net.IP) (bool, error) {
				s := ip.String()
				if _, ok := allocated[s]; ok {
					return false, nil
				}
				allocated[s] = struct{}{}
				return true, nil
			}

			storeFactory.CreateReturns(store, nil)
		})

		It("reserves an IP", func() {
			result, err := allocator.AllocateIP("network-id", "container-id")
			Expect(err).NotTo(HaveOccurred())

			By("creating the config for the network")
			Expect(configFactory.CreateCallCount()).To(Equal(1))
			Expect(configFactory.CreateArgsForCall(0)).To(Equal("network-id"))

			By("getting the store based on the network id")
			Expect(storeFactory.CreateCallCount()).To(Equal(1))
			Expect(storeFactory.CreateArgsForCall(0)).To(Equal("network-id"))

			By("skipping the gateway address gateway")
			Expect(allocated).NotTo(HaveKey("192.168.2.1"))
			Expect(result.IP4.Gateway.String()).To(Equal("192.168.2.1"))

			By("reserving the IP address")
			Expect(store.ReserveCallCount()).To(Equal(1))
			containerID, ipAddress := store.ReserveArgsForCall(0)
			Expect(containerID).To(Equal("container-id"))
			Expect(ipAddress.String()).To(Equal("192.168.2.2"))

			By("returning the result")
			Expect(result).To(Equal(&types.Result{
				IP4: &types.IPConfig{
					IP: net.IPNet{
						IP:   net.ParseIP("192.168.2.2").To4(),
						Mask: net.CIDRMask(24, 32),
					},
					Gateway: net.ParseIP("192.168.2.1").To4(),
				},
			}))
		})

		It("acquires and releases a lock when accessing stores collection", func() {
			allocator.AllocateIP("network-id", "container-id")

			Expect(storeLocker.LockCallCount()).To(Equal(1))
			Expect(storeLocker.UnlockCallCount()).To(Equal(1))
		})

		It("acquires and releases a lock when accessing the configs collection", func() {
			allocator.AllocateIP("network-id", "container-id")

			Expect(configLocker.LockCallCount()).To(Equal(1))
			Expect(configLocker.UnlockCallCount()).To(Equal(1))
		})

		Context("when a network configuration already exists", func() {
			It("does not create a new configuration", func() {
				allocator.AllocateIP("network-id", "container-id")
				allocator.AllocateIP("network-id", "container-id")

				Expect(configFactory.CreateCallCount()).To(Equal(1))
				Expect(configLocker.LockCallCount()).To(Equal(2))
				Expect(configLocker.UnlockCallCount()).To(Equal(2))
			})
		})

		Context("when the config factory fails to create a store", func() {
			BeforeEach(func() {
				configFactory.CreateReturns(ipam.Config{}, errors.New("network not found"))
			})

			It("returns a meaningful error", func() {
				_, err := allocator.AllocateIP("network-id", "container-id")
				Expect(configFactory.CreateCallCount()).To(Equal(1))
				Expect(err).To(MatchError(`failed to obtain configuration for network "network-id": network not found`))
			})
		})

		Context("when an allocator store already exists", func() {
			It("does not create a new allocator store", func() {
				allocator.AllocateIP("network-id", "container-id")
				allocator.AllocateIP("network-id", "container-id")

				Expect(storeFactory.CreateCallCount()).To(Equal(1))
				Expect(storeLocker.LockCallCount()).To(Equal(2))
				Expect(storeLocker.UnlockCallCount()).To(Equal(2))
			})
		})

		Context("when the store factory fails to create a store", func() {
			BeforeEach(func() {
				storeFactory.CreateReturns(nil, errors.New("out of disk space"))
			})

			It("returns a meaningful error", func() {
				_, err := allocator.AllocateIP("network-id", "container-id")
				Expect(storeFactory.CreateCallCount()).To(Equal(1))
				Expect(err).To(MatchError("failed to create allocator store: out of disk space"))
			})
		})

		Context("when the address space is exhausted", func() {
			BeforeEach(func() {
				store.ReserveReturns(false, nil)
			})

			It("returns a meaningful error", func() {
				_, err := allocator.AllocateIP("network-id", "container-id")
				Expect(err).To(Equal(ipam.NoMoreAddressesError))
			})
		})

		Context("when the store reservation fails", func() {
			BeforeEach(func() {
				store.ReserveReturns(false, errors.New("this is a problem"))
			})

			It("returns a meaningful error", func() {
				_, err := allocator.AllocateIP("network-id", "container-id")
				Expect(err).To(MatchError("failed to reserve IP: this is a problem"))
			})
		})

		Context("when a gateway is specified", func() {
			BeforeEach(func() {
				config.Gateway = net.ParseIP("192.168.2.3")
				configFactory.CreateReturns(config, nil)
			})

			It("uses the specified gateway", func() {
				result, err := allocator.AllocateIP("network-id", "container-id")
				Expect(err).NotTo(HaveOccurred())

				Expect(result.IP4.Gateway.String()).To(Equal("192.168.2.3"))
			})

			It("will not allocate an IP address that equals the gateway addresss", func() {
				var addresses []string

				for i := 0; i < 4; i++ {
					result, err := allocator.AllocateIP("network-id", "container-id")
					Expect(err).NotTo(HaveOccurred())

					addresses = append(addresses, result.IP4.IP.String())
				}

				Expect(addresses).To(Equal([]string{
					"192.168.2.1/24",
					"192.168.2.2/24",
					"192.168.2.4/24",
					"192.168.2.5/24",
				}))
			})
		})
	})
})
