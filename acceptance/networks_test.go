package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/config"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/network"
	"github.com/miekg/dns"

	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/vishvananda/netlink"    //only linux
	"github.com/vishvananda/netlink/nl" //only linux
)

func writeConfigFile(daemonConfig config.Daemon) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	daemonConfig.Marshal(configFile)
	Expect(configFile.Close()).To(Succeed())

	return configFile.Name()
}

var _ = Describe("Networks", func() {
	var (
		session     *gexec.Session
		address     string
		containerID string
		vni         int
		sandboxName string
		hostAddress string
		spaceID     string
		appID       string
		networkID   string

		sandboxRepo        namespace.Repository
		containerRepo      namespace.Repository
		containerNamespace namespace.Namespace
	)

	BeforeEach(func() {
		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())
		sandboxRepoDir, err := ioutil.TempDir("", "sandbox")
		Expect(err).NotTo(HaveOccurred())

		sandboxRepo, err = namespace.NewRepository(sandboxRepoDir)
		Expect(err).NotTo(HaveOccurred())

		containerRepoDir, err := ioutil.TempDir("", "containers")
		Expect(err).NotTo(HaveOccurred())

		containerRepo, err = namespace.NewRepository(containerRepoDir)
		Expect(err).NotTo(HaveOccurred())

		guid, err := uuid.NewV4()
		Expect(err).NotTo(HaveOccurred())

		containerNamespace, err = containerRepo.Create(guid.String())
		Expect(err).NotTo(HaveOccurred())

		hostAddress = "10.11.12.13"

		configFilePath := writeConfigFile(config.Daemon{
			ListenHost:        "127.0.0.1",
			ListenPort:        4001 + GinkgoParallelNode(),
			LocalSubnet:       "192.168.1.0/16",
			OverlayNetwork:    "192.168.0.0/16",
			SandboxDir:        sandboxRepoDir,
			Database:          testDatabase.DBConfig(),
			HostAddress:       hostAddress,
			OverlayDNSAddress: "192.168.255.254",
			ExternalDNSServer: "8.8.8.8",
			Suffix:            "potato",
		})

		ducatiCmd := exec.Command(ducatidPath, "-configFile", configFilePath)
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		// GinkgoParallelNode() necessary to avoid test pollution in parallel
		spaceID = fmt.Sprintf("some-space-id-%x", GinkgoParallelNode())
		networkID = spaceID
		containerID = fmt.Sprintf("some-container-id-%x", rand.Int())
		appID = fmt.Sprintf("some-app-id-%x", rand.Int())

		networkMapper := &network.FixedNetworkMapper{DefaultNetworkID: "default"}
		vni, err = networkMapper.GetVNI(spaceID)
		Expect(err).NotTo(HaveOccurred())

		sandboxName = fmt.Sprintf("vni-%d", vni)
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(containerRepo.Destroy(containerNamespace)).To(Succeed())
	})

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	It("should boot and gracefully terminate", func() {
		Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())

		Consistently(session).ShouldNot(gexec.Exit())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	})

	Describe("POST to /cni/add and /cni/del", func() {
		var (
			upSpec       models.CNIAddPayload
			downSpec     models.CNIDelPayload
			daemonClient *client.DaemonClient
			ipamResult   types.Result
		)

		BeforeEach(func() {
			Eventually(serverIsAvailable).Should(Succeed())

			daemonClient = client.New("http://"+address, http.DefaultClient)

			By("generating config and creating the request")
			upSpec = models.CNIAddPayload{
				Args:               "FOO=BAR;ABC=123",
				ContainerNamespace: containerNamespace.Name(),
				InterfaceName:      "vx-eth0",
				Network: models.NetworkPayload{
					models.Properties{
						AppID:   appID,
						SpaceID: spaceID,
					},
				},
				ContainerID: containerID,
			}

			downSpec = models.CNIDelPayload{
				InterfaceName:      "vx-eth0",
				ContainerNamespace: containerNamespace.Name(),
				ContainerID:        containerID,
			}

			By("adding the container to a network")
			var err error
			ipamResult, err = daemonClient.ContainerUp(upSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("removing the container from the network")
			Expect(daemonClient.ContainerDown(downSpec)).To(Succeed())

			By("checking that containers have been removed")
			containers, err := daemonClient.ListNetworkContainers(networkID)
			Expect(err).NotTo(HaveOccurred())
			Expect(containers).To(HaveLen(0))

			By("checking that the sandbox has been cleaned up")
			_, err = sandboxRepo.Get(sandboxName)
			Expect(err).To(MatchError(ContainSubstring("no such file or directory")))

			By("checking that the veth device is no longer in the container")
			err = containerNamespace.Execute(func(_ *os.File) error {
				_, err := netlink.LinkByName("vx-eth0")
				Expect(err).To(MatchError(ContainSubstring("Link not found")))
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("makes container metadata available on the list network containers endpoint", func() {
			containers, err := daemonClient.ListNetworkContainers(networkID)
			Expect(err).NotTo(HaveOccurred())

			Expect(containers).To(HaveLen(1))
			Expect(containers[0].HostIP).To(Equal(hostAddress))
		})

		It("makes container metadata available on the /containers endpoint", func() {
			containers, err := daemonClient.ListContainers()
			Expect(err).NotTo(HaveOccurred())

			for _, container := range containers {
				if container.ID != containerID {
					continue
				}
				Expect(container.SandboxName).To(Equal(sandboxName))
			}
		})

		It("makes container metadata available on get container endpoint", func() {
			container, err := daemonClient.GetContainer(containerID)
			Expect(err).NotTo(HaveOccurred())

			Expect(container.HostIP).To(Equal(hostAddress))
			Expect(container.NetworkID).To(Equal(networkID))
			Expect(container.App).To(Equal(appID))
		})

		Context("when the ADD endpoint is called a second time with the same container ID", func() {
			It("should return an error and not crash the system", func() {
				_, err := daemonClient.ContainerUp(upSpec) // 2nd time we're calling this
				Expect(err).To(MatchError(ipam.AlreadyOnNetworkError))
			})
		})

		It("brings up the loopback device in the sandbox", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())

			sandboxNS.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName("lo")
				Expect(err).NotTo(HaveOccurred())
				loopback, ok := link.(*netlink.Device)
				Expect(ok).To(BeTrue())

				Expect(loopback.LinkAttrs.Flags & net.FlagUp).To(Equal(net.FlagUp))

				return nil
			})
		})

		It("moves a vxlan adapter into the sandbox", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())

			sandboxNS.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName(fmt.Sprintf("vxlan%d", vni))
				Expect(err).NotTo(HaveOccurred())
				vxlan, ok := link.(*netlink.Vxlan)
				Expect(ok).To(BeTrue())

				Expect(vxlan.VxlanId).To(Equal(vni))
				Expect(vxlan.Learning).To(BeTrue())
				Expect(vxlan.Port).To(BeEquivalentTo(nl.Swap16(4789)))
				Expect(vxlan.Proxy).To(BeTrue())
				Expect(vxlan.L2miss).To(BeTrue())
				Expect(vxlan.L3miss).To(BeTrue())
				Expect(vxlan.LinkAttrs.Flags & net.FlagUp).To(Equal(net.FlagUp))

				return nil
			})
		})

		It("creates a vxlan bridge in the sandbox", func() {
			var bridge *netlink.Bridge
			var addrs []netlink.Addr

			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())

			err = sandboxNS.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName(fmt.Sprintf("vxlanbr%d", vni))
				if err != nil {
					return fmt.Errorf("finding link by name: %s", err)
				}

				var ok bool
				bridge, ok = link.(*netlink.Bridge)
				if !ok {
					return fmt.Errorf("unable to cast link to bridge")
				}

				addrs, err = netlink.AddrList(link, netlink.FAMILY_V4)
				if err != nil {
					return fmt.Errorf("unable to list addrs: %s", err)
				}

				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(bridge.LinkAttrs.MTU).To(Equal(1450))
			Expect(bridge.LinkAttrs.Flags & net.FlagUp).To(Equal(net.FlagUp))

			Expect(addrs).To(HaveLen(1))
			Expect(addrs[0].IPNet.IP.String()).To(Equal(ipamResult.IP4.Gateway.String()))
		})

		It("does not specify routes on the vxlan device", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())

			err = sandboxNS.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName(fmt.Sprintf("vxlan%d", vni))
				Expect(err).NotTo(HaveOccurred())
				vxlan, ok := link.(*netlink.Vxlan)
				Expect(ok).To(BeTrue())

				routes, err := netlink.RouteList(vxlan, netlink.FAMILY_V4)
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(BeEmpty())
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("has a route on the bridge for the overlay in the sandbox", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())

			err = sandboxNS.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName(fmt.Sprintf("vxlanbr%d", vni))
				Expect(err).NotTo(HaveOccurred())
				bridge, ok := link.(*netlink.Bridge)
				Expect(ok).To(BeTrue())

				routes, err := netlink.RouteList(bridge, netlink.FAMILY_V4)
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).NotTo(BeEmpty())

				var sanitizedRoutes []netlink.Route
				for _, route := range routes {
					sanitizedRoutes = append(sanitizedRoutes, netlink.Route{
						Gw:  route.Gw,
						Dst: route.Dst,
						Src: route.Src,
					})
				}

				_, vxlanNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(sanitizedRoutes).To(ConsistOf(netlink.Route{
					Src: net.ParseIP("192.168.1.1").To4(),
					Dst: vxlanNet,
				}))

				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("starts a DNS resolver in the sandbox at the configured overlay address", func() {
			By("resolving requests to external servers")
			err := containerNamespace.Execute(func(_ *os.File) error {
				client := dns.Client{
					Net: "udp",
				}
				message := &dns.Msg{
					MsgHdr: dns.MsgHdr{
						Id: dns.Id(),
					},
					Question: []dns.Question{
						dns.Question{
							Name:   dns.Fqdn("example.com"),
							Qtype:  dns.TypeA,
							Qclass: uint16(dns.ClassINET),
						},
					},
				}
				resp, _, err := client.Exchange(message, "192.168.255.254:53")
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Rcode).To(Equal(dns.RcodeSuccess))
				Expect(len(resp.Answer)).To(BeNumerically(">=", 1))
				_, ok := resp.Answer[0].(*dns.A)
				Expect(ok).To(BeTrue())

				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("resolving internal requests")
			err = containerNamespace.Execute(func(_ *os.File) error {
				client := dns.Client{
					Net: "udp",
				}
				message := &dns.Msg{
					MsgHdr: dns.MsgHdr{
						Id: dns.Id(),
					},
					Question: []dns.Question{
						dns.Question{
							Name:   dns.Fqdn(containerID + ".potato"),
							Qtype:  dns.TypeA,
							Qclass: uint16(dns.ClassINET),
						},
					},
				}
				resp, _, err := client.Exchange(message, "192.168.255.254:53")
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Rcode).To(Equal(dns.RcodeSuccess))
				Expect(len(resp.Answer)).To(BeNumerically(">=", 1))
				_, ok := resp.Answer[0].(*dns.A)
				Expect(ok).To(BeTrue())

				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates a veth pair in the container and sandbox namespaces", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())

			By("checking that the container has a veth device")
			err = containerNamespace.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName("vx-eth0")
				Expect(err).NotTo(HaveOccurred())

				bridge, ok := link.(*netlink.Veth)
				Expect(ok).To(BeTrue())
				Expect(bridge.LinkAttrs.MTU).To(Equal(1450))
				Expect(bridge.LinkAttrs.Flags & net.FlagUp).To(Equal(net.FlagUp))

				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the sandbox has a veth device")
			err = sandboxNS.Execute(func(_ *os.File) error {
				expectedLinkName := container.NameSandboxLink(containerID)
				link, err := netlink.LinkByName(expectedLinkName)
				Expect(err).NotTo(HaveOccurred())

				bridge, ok := link.(*netlink.Veth)
				Expect(ok).To(BeTrue())
				Expect(bridge.LinkAttrs.MTU).To(Equal(1450))
				Expect(bridge.LinkAttrs.Flags & net.FlagUp).To(Equal(net.FlagUp))

				return nil
			})
			Expect(err).NotTo(HaveOccurred())

		})

		It("defines routes in the container", func() {
			err := containerNamespace.Execute(func(_ *os.File) error {
				l, err := netlink.LinkByName("vx-eth0")
				Expect(err).NotTo(HaveOccurred())

				routes, err := netlink.RouteList(l, netlink.FAMILY_V4)
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))

				var sanitizedRoutes []netlink.Route
				for _, route := range routes {
					sanitizedRoutes = append(sanitizedRoutes, netlink.Route{
						Gw:  route.Gw,
						Dst: route.Dst,
						Src: route.Src,
					})
				}

				_, vxlanNet, err := net.ParseCIDR("192.168.0.0/16")
				Expect(sanitizedRoutes).To(ContainElement(netlink.Route{
					Src: net.ParseIP("192.168.1.2").To4(),
					Dst: vxlanNet,
				}))

				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
