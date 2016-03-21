package links_test

import (
	"errors"
	"io/ioutil"
	"net"
	"os"

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
		var expectedBridge *netlink.Bridge

		BeforeEach(func() {
			expectedBridge = &netlink.Bridge{
				LinkAttrs: netlink.LinkAttrs{
					Name: "some-bridge-name",
					MTU:  links.BridgeMTU,
				},
			}
		})

		It("adds the bridge", func() {
			err := factory.CreateBridge("some-bridge-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.LinkAddCallCount()).To(Equal(1))
			Expect(netlinker.LinkAddArgsForCall(0)).To(Equal(expectedBridge))
		})

		Context("when adding the bridge link fails", func() {
			It("returns the error", func() {
				netlinker.LinkAddReturns(errors.New("link add failed"))

				err := factory.CreateBridge("some-bridge-name")
				Expect(err).To(Equal(errors.New("link add failed")))
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

		It("should add the link", func() {
			err := factory.CreateVxlan("some-device-name", 42)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.LinkAddCallCount()).To(Equal(1))
			Expect(netlinker.LinkAddArgsForCall(0)).To(Equal(expectedVxlan))
		})

		Context("when adding the link fails", func() {
			It("should return the error", func() {
				netlinker.LinkAddReturns(errors.New("some error"))

				err := factory.CreateVxlan("some-device-name", 42)
				Expect(err).To(Equal(errors.New("some error")))
			})
		})

		Describe("CreateVeth", func() {
			It("adds a veth link with the appropriate names and MTU", func() {
				err := factory.CreateVeth("host", "peer", 999)
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkAddCallCount()).To(Equal(1))
				veth, ok := netlinker.LinkAddArgsForCall(0).(*netlink.Veth)
				Expect(ok).To(BeTrue())

				Expect(veth.Attrs().Name).To(Equal("host"))
				Expect(veth.Attrs().MTU).To(Equal(999))
				Expect(veth.PeerName).To(Equal("peer"))
			})

			Context("when adding the veth link fails", func() {
				BeforeEach(func() {
					netlinker.LinkAddReturns(errors.New("some error"))
				})

				It("returns the error", func() {
					err := factory.CreateVeth("host", "peer", 999)
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

		Describe("Exists", func() {
			Context("when a link is found", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(&netlink.Vxlan{VxlanId: 41}, nil)
				})

				It("returns true", func() {
					Expect(factory.Exists("some-device-name")).To(BeTrue())
					Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
					Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("some-device-name"))
				})
			})

			Context("when link by name returns an error", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(nil, errors.New("not found"))
				})

				It("returns false", func() {
					Expect(factory.Exists("some-device-name")).To(BeFalse())
					Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
					Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("some-device-name"))
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

		Describe("SetHardwareAddress", func() {
			var (
				hardwareAddr net.HardwareAddr
				foundLink    *netlink.Dummy
			)

			BeforeEach(func() {
				hardwareAddr, _ = net.ParseMAC("01:02:03:04:05:06")
				foundLink = &netlink.Dummy{}
				netlinker.LinkByNameReturns(foundLink, nil)
			})

			It("finds the link by name", func() {
				err := factory.SetHardwareAddress("link-name", hardwareAddr)
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
				Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("link-name"))
			})

			It("sets the hardware address of the link", func() {
				err := factory.SetHardwareAddress("link-name", hardwareAddr)
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkSetHardwareAddrCallCount()).To(Equal(1))
				link, hwAddr := netlinker.LinkSetHardwareAddrArgsForCall(0)
				Expect(link).To(BeIdenticalTo(foundLink))
				Expect(hwAddr).To(Equal(hardwareAddr))
			})

			Context("when finding the link fails", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(nil, errors.New("nope!"))
				})

				It("returns a meaningful error", func() {
					err := factory.SetHardwareAddress("link-name", hardwareAddr)
					Expect(err).To(MatchError("find link: nope!"))
				})
			})

			Context("when setting the hardware address fails", func() {
				BeforeEach(func() {
					netlinker.LinkSetHardwareAddrReturns(errors.New("boom!"))
				})

				It("returns a meaningful error", func() {
					err := factory.SetHardwareAddress("link-name", hardwareAddr)
					Expect(err).To(MatchError("set hw addr: boom!"))
				})
			})
		})

		Describe("SetMaster", func() {
			var master, slave netlink.Link
			var masterErr, slaveErr error

			BeforeEach(func() {
				master = &netlink.Bridge{}
				slave = &netlink.Dummy{}
				masterErr = nil
				slaveErr = nil

				netlinker.LinkByNameStub = func(name string) (netlink.Link, error) {
					switch name {
					case "master":
						return master, masterErr
					case "slave":
						return slave, slaveErr
					default:
						return nil, errors.New("unknown")
					}
				}
			})

			It("sets the slave's master", func() {
				err := factory.SetMaster("slave", "master")
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkByNameCallCount()).To(Equal(2))
				Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("master"))
				Expect(netlinker.LinkByNameArgsForCall(1)).To(Equal("slave"))

				Expect(netlinker.LinkSetMasterCallCount()).To(Equal(1))
				s, m := netlinker.LinkSetMasterArgsForCall(0)
				Expect(s).To(Equal(slave))
				Expect(m).To(Equal(master))
			})

			Context("when finding the master link fails", func() {
				BeforeEach(func() {
					masterErr = errors.New("potato")
				})

				It("wraps and returns the error", func() {
					err := factory.SetMaster("slave", "master")
					Expect(err).To(MatchError("failed to find master: potato"))
				})
			})

			Context("when the master is not a bridge", func() {
				BeforeEach(func() {
					master = &netlink.Dummy{}
				})

				It("returns a meaningful error", func() {
					err := factory.SetMaster("slave", "master")
					Expect(err).To(MatchError("master must be a bridge"))
				})
			})

			Context("when finding the slave link fails", func() {
				BeforeEach(func() {
					slaveErr = errors.New("po-tah-toe")
				})

				It("wraps and returns the error", func() {
					err := factory.SetMaster("slave", "master")
					Expect(err).To(MatchError("failed to find slave: po-tah-toe"))
				})

			})

			Context("when enslaving fails", func() {
				BeforeEach(func() {
					netlinker.LinkSetMasterReturns(errors.New("you're not a slave"))
				})

				It("returns a meaningful error", func() {
					err := factory.SetMaster("slave", "master")
					Expect(err).To(MatchError("failed to set master: you're not a slave"))
				})
			})
		})

		Describe("SetNamespace", func() {
			var link netlink.Link
			var namespacePath string

			BeforeEach(func() {
				link = &netlink.Dummy{}
				netlinker.LinkByNameReturns(link, nil)

				tempFile, err := ioutil.TempFile("", "set-namespace")
				Expect(err).NotTo(HaveOccurred())

				_, err = tempFile.Write([]byte("hello"))
				Expect(err).NotTo(HaveOccurred())

				namespacePath = tempFile.Name()
				tempFile.Close()
			})

			It("associates the link with a file descriptor", func() {
				netlinker.LinkSetNsFdStub = func(l netlink.Link, fd int) error {
					Expect(l).To(Equal(link))

					f := os.NewFile(uintptr(fd), "name")
					payload, err := ioutil.ReadAll(f)
					Expect(err).NotTo(HaveOccurred())
					Expect(payload).To(Equal([]byte("hello")))

					return nil
				}

				err := factory.SetNamespace("link-name", namespacePath)
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
				Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("link-name"))

				Expect(netlinker.LinkSetNsFdCallCount()).To(Equal(1))
			})

			Context("when finding the link fails", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(nil, errors.New("no link for you"))
				})

				It("returns a meaningful error", func() {
					err := factory.SetNamespace("link-name", namespacePath)
					Expect(err).To(MatchError("failed to find link: no link for you"))
				})
			})

			Context("when opening the namespace fails", func() {
				BeforeEach(func() {
					namespacePath = "some junk path name that is bogus"
				})

				It("returns a meaningful error", func() {
					err := factory.SetNamespace("link-name", namespacePath)
					Expect(err).To(MatchError(HavePrefix("failed to open namespace: open")))
				})
			})

			Context("when assigning the namespace fails", func() {
				BeforeEach(func() {
					netlinker.LinkSetNsFdReturns(errors.New("what namespace?"))
				})

				It("returns a meaningful error", func() {
					err := factory.SetNamespace("link-name", namespacePath)
					Expect(err).To(MatchError("failed to set link namespace: what namespace?"))
				})
			})
		})

		Describe("HardwareAddress", func() {
			var (
				expectedHwAddr net.HardwareAddr
				link           *netlink.Dummy
			)

			BeforeEach(func() {
				expectedHwAddr, _ = net.ParseMAC("01:02:03:04:05:06")
				link = &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						HardwareAddr: expectedHwAddr,
					},
				}
				netlinker.LinkByNameReturns(link, nil)
			})

			It("returns the hardware address of the named link", func() {
				hwAddr, err := factory.HardwareAddress("link-name")
				Expect(err).NotTo(HaveOccurred())

				Expect(hwAddr).To(Equal(expectedHwAddr))
			})

			Context("when finding the link fails", func() {
				BeforeEach(func() {
					netlinker.LinkByNameReturns(nil, errors.New("no link for you"))
				})

				It("returns a meaningful error", func() {
					_, err := factory.HardwareAddress("link-name")
					Expect(err).To(MatchError("failed to find link: no link for you"))
				})
			})
		})

		Describe("SetUp", func() {
			var link netlink.Link

			BeforeEach(func() {
				link = &netlink.Dummy{}
				netlinker.LinkByNameReturns(link, nil)
			})

			It("sets the link up", func() {
				err := factory.SetUp("my-link")
				Expect(err).NotTo(HaveOccurred())

				Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
				Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("my-link"))

				Expect(netlinker.LinkSetUpCallCount()).To(Equal(1))
				Expect(netlinker.LinkSetUpArgsForCall(0)).To(Equal(link))
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

			Context("when setting the link up fails", func() {
				BeforeEach(func() {
					netlinker.LinkSetUpReturns(errors.New("I've fallen and I can't get up"))
				})

				It("returns a meaningful error", func() {
					err := factory.SetUp("my-link")
					Expect(err).To(MatchError("failed to set link up: I've fallen and I can't get up"))
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

		Describe("VethDeviceCount", func() {
			var link1, link2, link3 netlink.Link

			BeforeEach(func() {
				link1 = &netlink.Dummy{}
				link2 = &netlink.Veth{}
				link3 = &netlink.Veth{}

				netlinker.LinkListReturns([]netlink.Link{link1, link2, link3}, nil)
			})

			It("returns the number of veth devices", func() {
				count, err := factory.VethDeviceCount()
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(2))
			})

			Context("when there are no veth devices", func() {
				BeforeEach(func() {
					netlinker.LinkListReturns([]netlink.Link{link1}, nil)
				})

				It("succeeds and returns 0", func() {
					count, err := factory.VethDeviceCount()
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(0))
				})
			})

			Context("when listing links fails", func() {
				BeforeEach(func() {
					netlinker.LinkListReturns(nil, errors.New("banana"))
				})

				It("wraps and returns the error", func() {
					_, err := factory.VethDeviceCount()
					Expect(err).To(MatchError("failed to list links: banana"))
				})
			})

		})
	})
})
