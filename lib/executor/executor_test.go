package executor_test

import (
	"errors"
	"fmt"
	"net"

	"golang.org/x/sys/unix"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/executor/fakes"
	nl_fakes "github.com/cloudfoundry-incubator/ducati-daemon/lib/nl/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/ns"
	ns_fakes "github.com/cloudfoundry-incubator/ducati-daemon/lib/ns/fakes"

	"github.com/vishvananda/netlink"

	"github.com/appc/cni/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type TestLink struct {
	Attributes netlink.LinkAttrs
}

func (t TestLink) Attrs() *netlink.LinkAttrs {
	return &t.Attributes
}

func (t TestLink) Type() string {
	return "NOT IMPLEMENTED"
}

var _ = Describe("SetupContainerNS", func() {
	var (
		ex                executor.Executor
		networkNamespacer *ns_fakes.Namespacer
		linkFactory       *fakes.LinkFactory
		netlinker         *nl_fakes.Netlinker
		addressManager    *fakes.AddressManager

		sandboxNsHandle   *ns_fakes.Handle
		containerNsHandle *ns_fakes.Handle
		hostHandle        *ns_fakes.Handle

		sandboxFd uintptr

		returnedSandboxLink   netlink.Link
		returnedContainerLink netlink.Link
		result                types.Result
	)

	BeforeEach(func() {
		networkNamespacer = &ns_fakes.Namespacer{}
		linkFactory = &fakes.LinkFactory{}
		netlinker = &nl_fakes.Netlinker{}
		addressManager = &fakes.AddressManager{}

		ex = executor.Executor{
			NetworkNamespacer: networkNamespacer,
			LinkFactory:       linkFactory,
			Netlinker:         netlinker,
			AddressManager:    addressManager,
		}

		sandboxFd = 9999
		sandboxNsHandle = &ns_fakes.Handle{}
		sandboxNsHandle.FdReturns(sandboxFd)

		containerNsHandle = &ns_fakes.Handle{}
		hostHandle = &ns_fakes.Handle{}

		networkNamespacer.GetFromPathStub = func(ns string) (ns.Handle, error) {
			switch ns {
			case "/var/some/sandbox/namespace":
				return sandboxNsHandle, nil
			case "/var/some/container/namespace":
				return containerNsHandle, nil
			case "/proc/self/ns/net":
				return hostHandle, nil
			default:
				return &ns_fakes.Handle{}, nil
			}
		}

		hwAddr, err := net.ParseMAC("ff:ff:ff:ff:ff:ff")
		Expect(err).NotTo(HaveOccurred())

		returnedSandboxLink = TestLink{Attributes: netlink.LinkAttrs{Name: "some-contai"}}
		returnedContainerLink = TestLink{Attributes: netlink.LinkAttrs{
			Index:        1555,
			Name:         "some-eth0",
			HardwareAddr: hwAddr,
		}}

		linkFactory.CreateVethPairReturns(nil)

		linkFactory.FindLinkStub = func(name string) (netlink.Link, error) {
			switch name {
			case "some-contai":
				return returnedSandboxLink, nil
			case "some-eth0":
				return returnedContainerLink, nil
			default:
				return nil, fmt.Errorf("unknown link: %q", name)
			}
		}

		result = types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("192.168.100.1"),
					Mask: net.ParseIP("192.168.100.1").DefaultMask(),
				},
				Gateway: net.ParseIP("192.168.100.1"),
				Routes: []types.Route{
					{
						Dst: net.IPNet{
							IP:   net.ParseIP("192.168.1.5"),
							Mask: net.ParseIP("192.168.1.5").DefaultMask(),
						},
						GW: net.ParseIP("192.168.1.1"),
					},
				},
			},
		}
	})

	It("should construct the network inside the container namespace", func() {
		sandboxLink, containerMAC, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)
		Expect(err).NotTo(HaveOccurred())

		By("asking for the host namespace handle")
		Expect(networkNamespacer.GetFromPathCallCount()).To(Equal(3))
		Expect(networkNamespacer.GetFromPathArgsForCall(0)).To(Equal("/proc/self/ns/net"))

		By("asking for the container namespace handle")
		Expect(networkNamespacer.GetFromPathArgsForCall(1)).To(Equal("/var/some/container/namespace"))

		By("switch to the container namespace via the handle")
		Expect(networkNamespacer.SetCallCount()).To(Equal(2))
		Expect(networkNamespacer.SetArgsForCall(0)).To(Equal(containerNsHandle))

		By("creating a veth pair when the container namespace")
		Expect(linkFactory.CreateVethPairCallCount()).To(Equal(1))
		containerID, interfaceName, vxlanVethMTU := linkFactory.CreateVethPairArgsForCall(0)
		Expect(containerID).To(Equal("some-contai"))
		Expect(interfaceName).To(Equal("some-eth0"))
		Expect(vxlanVethMTU).To(Equal(1450))

		By("finding the container link")
		Expect(linkFactory.FindLinkArgsForCall(0)).To(Equal("some-eth0"))

		By("finding the sandbox link")
		Expect(linkFactory.FindLinkArgsForCall(1)).To(Equal("some-contai"))

		By("getting the sandbox namespace")
		Expect(networkNamespacer.GetFromPathArgsForCall(2)).To(Equal("/var/some/sandbox/namespace"))

		By("moving the sandboxlink into the sandbox namespace")
		Expect(netlinker.LinkSetNsFdCallCount()).To(Equal(1))
		sandboxLink, fd := netlinker.LinkSetNsFdArgsForCall(0)
		Expect(sandboxLink).To(Equal(returnedSandboxLink))
		Expect(fd).To(BeEquivalentTo(sandboxFd))

		By("adding an address to the container link")
		Expect(addressManager.AddAddressCallCount()).To(Equal(1))
		name, returnedResult := addressManager.AddAddressArgsForCall(0)
		Expect(name).To(Equal("some-eth0"))
		Expect(returnedResult).To(Equal(&result.IP4.IP))

		By("setting the container link to UP")
		Expect(netlinker.LinkSetUpCallCount()).To(Equal(1))
		Expect(netlinker.LinkSetUpArgsForCall(0)).To(Equal(returnedContainerLink))

		By("refreshing the containerlink to get its hardware address")
		Expect(linkFactory.FindLinkCallCount()).To(Equal(3))
		Expect(linkFactory.FindLinkArgsForCall(2)).To(Equal("some-eth0"))

		By("adding a route")
		Expect(netlinker.RouteAddCallCount()).To(Equal(1))
		route := netlinker.RouteAddArgsForCall(0)
		Expect(route.LinkIndex).To(Equal(1555))
		Expect(route.Scope).To(Equal(netlink.SCOPE_UNIVERSE))
		Expect(route.Dst).To(Equal(&result.IP4.Routes[0].Dst))
		Expect(route.Gw).To(Equal(result.IP4.Routes[0].GW))

		By("setting namespace back to host namespace")
		Expect(networkNamespacer.SetArgsForCall(1)).To(Equal(hostHandle))

		By("closing the handles")
		Expect(hostHandle.CloseCallCount()).To(Equal(1))
		Expect(sandboxNsHandle.CloseCallCount()).To(Equal(1))
		Expect(containerNsHandle.CloseCallCount()).To(Equal(1))

		By("verifying return link and containermac")
		Expect(sandboxLink.Attrs().Name).To(Equal("some-contai"))
		Expect(containerMAC).To(Equal("ff:ff:ff:ff:ff:ff"))
	})

	Context("when no routes are specified", func() {
		BeforeEach(func() {
			result.IP4.Routes = []types.Route{}
		})

		It("does not attempt to add routes", func() {
			Expect(netlinker.RouteAddCallCount()).To(Equal(0))
		})
	})

	Context("when multiple routes are specified", func() {
		BeforeEach(func() {
			result.IP4.Routes = append(result.IP4.Routes, types.Route{
				Dst: net.IPNet{
					IP:   net.ParseIP("10.10.10.10"),
					Mask: net.CIDRMask(8, 32),
				},
				GW: net.ParseIP("10.10.10.1"),
			})
		})

		It("adds all routes", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.RouteAddCallCount()).To(Equal(2))

			route := netlinker.RouteAddArgsForCall(0)
			Expect(route.LinkIndex).To(Equal(1555))
			Expect(route.Scope).To(Equal(netlink.SCOPE_UNIVERSE))
			Expect(route.Dst).To(Equal(&result.IP4.Routes[0].Dst))
			Expect(route.Gw).To(Equal(result.IP4.Routes[0].GW))

			route = netlinker.RouteAddArgsForCall(1)
			Expect(route.LinkIndex).To(Equal(1555))
			Expect(route.Scope).To(Equal(netlink.SCOPE_UNIVERSE))
			Expect(route.Dst).To(Equal(&result.IP4.Routes[1].Dst))
			Expect(route.Gw).To(Equal(result.IP4.Routes[1].GW))
		})
	})

	Context("When a gateway is missing from the the route", func() {
		BeforeEach(func() {
			result.IP4.Routes[0].GW = nil
		})

		It("uses the default gateway for the route", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.RouteAddCallCount()).To(Equal(1))

			route := netlinker.RouteAddArgsForCall(0)
			Expect(route.LinkIndex).To(Equal(1555))
			Expect(route.Scope).To(Equal(netlink.SCOPE_UNIVERSE))
			Expect(route.Dst).To(Equal(&result.IP4.Routes[0].Dst))
			Expect(route.Gw).To(Equal(result.IP4.Gateway))
		})
	})

	Context("when getting the host namespace fails", func() {
		BeforeEach(func() {
			networkNamespacer.GetFromPathReturns(nil, errors.New("can't find my own namespace"))
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`failed to get host namespace handle: can't find my own namespace`))
		})
	})

	Context("when opening the container namespace fails", func() {
		BeforeEach(func() {
			networkNamespacer.GetFromPathStub = func(path string) (ns.Handle, error) {
				if path == "/var/some/container/namespace" {
					return nil, errors.New("failed to open")
				}

				return hostHandle, nil
			}
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`could not open container namespace "/var/some/container/namespace": failed to open`))
		})
	})

	Context("when setting the namespace fails", func() {
		BeforeEach(func() {
			networkNamespacer.SetStub = func(handle ns.Handle) error {
				if handle == containerNsHandle {
					return errors.New("original set error")
				}
				return nil
			}
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`set container namespace "/var/some/container/namespace" failed: original set error`))
		})

		It("closes the container namespace handle", func() {
			ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(containerNsHandle.CloseCallCount()).To(Equal(1))
		})
	})

	Context("when creating the veth pair fails", func() {
		BeforeEach(func() {
			linkFactory.CreateVethPairReturns(errors.New("nobody wants a veth"))
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`could not create veth pair: nobody wants a veth`))
		})
	})

	Context("when finding the container link fails", func() {
		BeforeEach(func() {
			linkFactory.FindLinkReturns(nil, errors.New("some error"))
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`could not get container link: some error`))
		})
	})

	Context("when finding the sandbox link fails", func() {
		BeforeEach(func() {
			linkFactory.FindLinkStub = func(name string) (netlink.Link, error) {
				if linkFactory.FindLinkCallCount() == 2 {
					return nil, errors.New("some error")
				}
				return nil, nil
			}
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`could not get sandbox link: some error`))
		})
	})

	Context("when getting the sandbox namespace handle fails", func() {
		BeforeEach(func() {
			networkNamespacer.GetFromPathStub = func(ns string) (ns.Handle, error) {
				switch ns {
				case "/var/some/container/namespace":
					return containerNsHandle, nil
				case "/proc/self/ns/net":
					return hostHandle, nil
				default:
					return &ns_fakes.Handle{}, errors.New("wow, a failure")
				}
			}
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`failed to get sandbox namespace handle: wow, a failure`))
		})
	})

	Context("when moving the sandbox link into the sandbox fails", func() {
		BeforeEach(func() {
			netlinker.LinkSetNsFdReturns(errors.New("boom"))
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`failed to move sandbox link into sandbox: boom`))
		})

		It("closes the sandbox namespace handle", func() {
			ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(sandboxNsHandle.CloseCallCount()).To(Equal(1))
		})
	})

	Context("when setting the address on the container link fails", func() {
		BeforeEach(func() {
			addressManager.AddAddressReturns(errors.New("no address for you"))
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`setting container address failed: no address for you`))
		})
	})

	Context("when setting the container link UP fails", func() {
		BeforeEach(func() {
			netlinker.LinkSetUpReturns(errors.New("explosivo"))
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`failed to up container link: explosivo`))
		})
	})

	Context("when refreshing the container link fails", func() {
		BeforeEach(func() {
			linkFactory.FindLinkStub = func(name string) (netlink.Link, error) {
				if linkFactory.FindLinkCallCount() == 3 {
					return nil, errors.New("some error")
				}

				switch name {
				case "some-contai":
					return returnedSandboxLink, nil
				case "some-eth0":
					return returnedContainerLink, nil
				default:
					return nil, fmt.Errorf("unknown link: %q", name)
				}
			}
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`failed to refresh container link: some error`))
		})
	})

	Context("when adding a route fails with EEXIST", func() {
		BeforeEach(func() {
			result.IP4.Routes = append(result.IP4.Routes, types.Route{
				Dst: net.IPNet{
					IP:   net.ParseIP("10.10.10.10"),
					Mask: net.CIDRMask(8, 32),
				},
				GW: net.ParseIP("10.10.10.1"),
			})

			netlinker.RouteAddStub = func(*netlink.Route) error {
				if netlinker.RouteAddCallCount() == 1 {
					return unix.EEXIST
				}
				return nil
			}
		})

		It("proceeds to the next route without failing", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).NotTo(HaveOccurred())
			Expect(netlinker.RouteAddCallCount()).To(Equal(2))
		})
	})

	Context("when adding routes fails with something other than EEXIST", func() {
		BeforeEach(func() {
			netlinker.RouteAddReturns(errors.New("invalid destination"))
		})

		It("wraps the error with a helpful message", func() {
			_, _, err := ex.SetupContainerNS("/var/some/sandbox/namespace", "/var/some/container/namespace", "some-container-id", "some-eth0", result)

			Expect(err).To(MatchError(`adding route to 192.168.1.5/24 via 192.168.1.1 failed: invalid destination`))
		})
	})
})
