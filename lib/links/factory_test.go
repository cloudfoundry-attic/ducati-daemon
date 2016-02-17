package links_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/links"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl/fakes"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Factory", func() {
	var (
		factory   *links.Factory
		netlinker *fakes.Netlinker
	)

	BeforeEach(func() {
		netlinker = &fakes.Netlinker{}
		factory = &links.Factory{
			Netlinker: netlinker,
		}
	})

	Describe("CreateBridge", func() {
		var (
			expectedBridge *netlink.Bridge
			address        *net.IPNet
		)

		BeforeEach(func() {
			var err error
			expectedBridge = &netlink.Bridge{
				LinkAttrs: netlink.LinkAttrs{
					Name: "some-bridge-name",
					MTU:  links.BridgeMTU,
				},
			}

			_, address, err = net.ParseCIDR("192.168.1.1/24")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return a bridge with the expected config", func() {
			bridge, err := factory.CreateBridge("some-bridge-name", address)
			Expect(err).NotTo(HaveOccurred())
			Expect(bridge).To(Equal(expectedBridge))
		})

		It("adds the bridge", func() {
			_, err := factory.CreateBridge("some-bridge-name", address)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.LinkAddCallCount()).To(Equal(1))
			Expect(netlinker.LinkAddArgsForCall(0)).To(Equal(expectedBridge))
		})

		Context("when adding the bridge link fails", func() {
			It("returns the error", func() {
				netlinker.LinkAddReturns(errors.New("link add failed"))

				_, err := factory.CreateBridge("some-bridge-name", address)
				Expect(err).To(Equal(errors.New("link add failed")))
			})
		})

		It("assigns the specified address", func() {
			_, err := factory.CreateBridge("some-bridge-name", address)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.AddrAddCallCount()).To(Equal(1))
			br, addr := netlinker.AddrAddArgsForCall(0)
			Expect(br).To(Equal(expectedBridge))
			Expect(addr).To(Equal(&netlink.Addr{IPNet: address}))
		})

		Context("when assigning an address to the bridge fails", func() {
			It("returns the error", func() {
				netlinker.AddrAddReturns(errors.New("addr add failed"))

				_, err := factory.CreateBridge("some-bridge-name", address)
				Expect(err).To(Equal(errors.New("addr add failed")))
			})
		})

		It("sets the bridge link up", func() {
			_, err := factory.CreateBridge("some-bridge-name", address)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.LinkSetUpCallCount()).To(Equal(1))
			Expect(netlinker.LinkSetUpArgsForCall(0)).To(Equal(expectedBridge))
		})

		Context("when setting the bridge link up fails", func() {
			It("returns the error", func() {
				netlinker.LinkSetUpReturns(errors.New("bridge link up failed"))

				_, err := factory.CreateBridge("some-bridge-name", address)
				Expect(err).To(Equal(errors.New("bridge link up failed")))
			})
		})
	})

	Describe("CreateVxlan", func() {
		var expectedVxlan *netlink.Vxlan

		BeforeEach(func() {
			expectedVxlan = &netlink.Vxlan{
				LinkAttrs: netlink.LinkAttrs{
					Name: "some-device-name",
				},
				VxlanId:  int(42),
				Learning: true,
				Port:     int(nl.Swap16(links.VxlanPort)), //network endian order
				Proxy:    true,
				L3miss:   true,
				L2miss:   true,
			}
		})

		It("should return a vxlan with the expected config", func() {
			link, err := factory.CreateVxlan("some-device-name", 42)
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(expectedVxlan))
		})

		It("should add the link", func() {
			_, err := factory.CreateVxlan("some-device-name", 42)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.LinkAddCallCount()).To(Equal(1))
			Expect(netlinker.LinkAddArgsForCall(0)).To(Equal(expectedVxlan))
		})

		Context("when adding the link fails", func() {
			It("should return the error", func() {
				netlinker.LinkAddReturns(errors.New("some error"))

				_, err := factory.CreateVxlan("some-device-name", 42)
				Expect(err).To(Equal(errors.New("some error")))
			})
		})

		Describe("CreateVethPair", func() {
			It("adds a veth link with the appropriate names and MTU", func() {
				err := factory.CreateVethPair("container", "host", 999)
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkAddCallCount()).To(Equal(1))
				veth, ok := netlinker.LinkAddArgsForCall(0).(*netlink.Veth)
				Expect(ok).To(BeTrue())

				Expect(veth.Attrs().Name).To(Equal("host"))
				Expect(veth.Attrs().MTU).To(Equal(999))
				Expect(veth.PeerName).To(Equal("container"))
			})

			Context("when adding the veth link fails", func() {
				BeforeEach(func() {
					netlinker.LinkAddReturns(errors.New("some error"))
				})

				It("returns the error", func() {
					err := factory.CreateVethPair("container", "host", 999)
					Expect(err).To(MatchError("link add: some error"))
				})
			})
		})

		Describe("FindLink", func() {
			Context("when a link is found", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(&netlink.Vxlan{VxlanId: 41}, nil)
				})

				It("should return the link", func() {
					link, err := factory.FindLink("some-device-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(link).To(Equal(&netlink.Vxlan{VxlanId: 41}))
				})
			})

			Context("when the link does not exist", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(nil, errors.New("not found"))
				})

				It("should return nil", func() {
					_, err := factory.FindLink("some-device-name")
					Expect(err).To(Equal(errors.New("not found")))
				})
			})
		})

		Describe("DeleteLink", func() {
			It("deletes the link", func() {
				link := &netlink.Dummy{}

				err := factory.DeleteLink(link)
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkDelCallCount()).To(Equal(1))
				Expect(netlinker.LinkDelArgsForCall(0)).To(Equal(link))
			})

			Context("when netlink LinkDel fails", func() {
				BeforeEach(func() {
					netlinker.LinkDelReturns(errors.New("link del failed"))
				})

				It("returns the error", func() {
					link := &netlink.Dummy{}
					err := factory.DeleteLink(link)
					Expect(err).To(MatchError("link del failed"))
				})
			})
		})

		Describe("DeleteLinkByName", func() {
			var expectedLink netlink.Link

			BeforeEach(func() {
				expectedLink = &netlink.Dummy{}
				netlinker.LinkByNameReturns(expectedLink, nil)
			})

			It("finds the link by name before deleting it", func() {
				err := factory.DeleteLinkByName("test-link")
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
				Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("test-link"))

				Expect(netlinker.LinkDelCallCount()).To(Equal(1))
				Expect(netlinker.LinkDelArgsForCall(0)).To(Equal(expectedLink))
			})

			Context("when finding the link fails", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(nil, errors.New("can't find it"))
				})

				It("returns the error", func() {
					err := factory.DeleteLinkByName("test-link")
					Expect(err).To(MatchError("can't find it"))
				})
			})

			Context("when deleting the link fails", func() {
				BeforeEach(func() {
					netlinker.LinkDelReturns(errors.New("delete failed"))
				})

				It("returns the error", func() {
					err := factory.DeleteLinkByName("test-link")
					Expect(err).To(MatchError("delete failed"))
				})
			})
		})

		Describe("ListLinks", func() {
			var link1, link2 netlink.Link

			BeforeEach(func() {
				link1 = &netlink.Dummy{}
				link2 = &netlink.Veth{}

				netlinker.LinkListReturns([]netlink.Link{link1, link2}, nil)
			})

			It("returns the links", func() {
				links, err := factory.ListLinks()
				Expect(err).NotTo(HaveOccurred())
				Expect(links).To(ConsistOf(link1, link2))
			})

			Context("when listing links fails", func() {
				BeforeEach(func() {
					netlinker.LinkListReturns(nil, errors.New("list links failed"))
				})

				It("returns the error", func() {
					_, err := factory.ListLinks()
					Expect(err).To(MatchError("list links failed"))
				})
			})
		})
	})
})
