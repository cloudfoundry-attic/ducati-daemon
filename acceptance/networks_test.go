package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/vishvananda/netlink"    //only linux
	"github.com/vishvananda/netlink/nl" //only linux
)

var _ = Describe("Networks", func() {
	var (
		session     *gexec.Session
		address     string
		networkID   string
		containerID string
		vni         int
		sandboxName string

		sandboxRepo        namespace.Repository
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

		containerRepo, err := namespace.NewRepository(containerRepoDir)
		Expect(err).NotTo(HaveOccurred())

		guid, err := uuid.NewV4()
		Expect(err).NotTo(HaveOccurred())

		containerNamespace, err = containerRepo.Create(guid.String())
		Expect(err).NotTo(HaveOccurred())

		Expect(err).NotTo(HaveOccurred())

		ducatiCmd := exec.Command(
			ducatidPath,
			"-listenAddr", address,
			"-overlayNetwork", "192.168.0.0/16",
			"-localSubnet", "192.168.99.0/24",
			"-databaseURL", testDatabase.URL(),
			"-sandboxRepoDir", sandboxRepoDir,
		)
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		networkID = fmt.Sprintf("some-network-id-%x", rand.Int())
		containerID = fmt.Sprintf("some-container-id-%x", rand.Int())
		vni = GinkgoParallelNode() // necessary to avoid test pollution in parallel
		sandboxName = fmt.Sprintf("vni-%d", vni)
	})

	AfterEach(func() {
		session.Kill()
		Eventually(session).Should(gexec.Exit())
		containerNamespace.Destroy()
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

	Describe("POST and DELETE /networks/:network_id/:container_id", func() {
		var (
			createURL     string
			deleteURL     string
			payload       []byte
			deletePayload []byte
			ipamResult    types.Result
		)

		BeforeEach(func() {
			Eventually(serverIsAvailable).Should(Succeed())

			By("generating config and creating the request")
			ipamResult = types.Result{
				IP4: &types.IPConfig{
					IP: net.IPNet{
						IP:   net.ParseIP("192.168.1.2"),
						Mask: net.CIDRMask(24, 32),
					},
					Gateway: net.ParseIP("192.168.1.1"),
					Routes: []types.Route{{
						Dst: net.IPNet{
							IP:   net.ParseIP("192.168.0.0"),
							Mask: net.CIDRMask(16, 32),
						},
						GW: net.ParseIP("192.168.1.1"),
					}},
				},
			}

			_, destination, err := net.ParseCIDR("10.10.10.0/24")
			Expect(err).NotTo(HaveOccurred())

			ipamResult.IP4.Routes = append(ipamResult.IP4.Routes, types.Route{Dst: *destination})

			payload, err = json.Marshal(models.NetworksSetupContainerPayload{
				Args:               "FOO=BAR;ABC=123",
				ContainerNamespace: containerNamespace.Path(),
				InterfaceName:      "vx-eth0",
				VNI:                vni,
				IPAM:               ipamResult,
			})
			Expect(err).NotTo(HaveOccurred())

			deletePayload, err = json.Marshal(models.NetworksDeleteContainerPayload{
				InterfaceName:      "vx-eth0",
				ContainerNamespace: containerNamespace.Path(),
				VNI:                vni,
			})
			Expect(err).NotTo(HaveOccurred())

			createURL = fmt.Sprintf("http://%s/networks/%s/%s", address, networkID, containerID)
			deleteURL = createURL

			By("POSTing to the endpoint")
			req, err := http.NewRequest("POST", createURL, bytes.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
		})

		It("should respond to POST and DELETE /networks/:network_id/:container_id", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

			By("getting the newly created container")
			listURL := fmt.Sprintf("http://%s/networks/%s", address, networkID)
			resp, err := http.Get(listURL)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			jsonBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var containers []models.Container
			err = json.Unmarshal(jsonBytes, &containers)
			Expect(err).NotTo(HaveOccurred())
			Expect(containers).To(HaveLen(1))

			By("deleting the container")
			req, err := http.NewRequest("DELETE", deleteURL, bytes.NewReader(deletePayload))
			Expect(err).NotTo(HaveOccurred())

			resp, err = http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			_, err = sandboxRepo.Get(sandboxName)
			Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
		})

		It("moves a vxlan adapter into the sandbox", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

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

			By("deleting")
			req, err := http.NewRequest("DELETE", deleteURL, bytes.NewReader(deletePayload))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
		})

		It("creates a vxlan bridge in the sandbox", func() {
			var bridge *netlink.Bridge
			var addrs []netlink.Addr

			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

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

			By("deleting")
			req, err := http.NewRequest("DELETE", deleteURL, bytes.NewReader(deletePayload))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
		})

		It("creates a veth pair in the container and sandbox namespaces", func() {
			sandboxNS, err := sandboxRepo.Get(sandboxName)
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

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
				link, err := netlink.LinkByName("some-container-")
				Expect(err).NotTo(HaveOccurred())

				bridge, ok := link.(*netlink.Veth)
				Expect(ok).To(BeTrue())
				Expect(bridge.LinkAttrs.MTU).To(Equal(1450))
				Expect(bridge.LinkAttrs.Flags & net.FlagUp).To(Equal(net.FlagUp))

				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("deleting")
			req, err := http.NewRequest("DELETE", deleteURL, bytes.NewReader(deletePayload))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			By("checking that the veth device is no longer in the container")
			err = containerNamespace.Execute(func(_ *os.File) error {
				_, err := netlink.LinkByName("vx-eth0")
				Expect(err).To(MatchError(ContainSubstring("Link not found")))
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there are routes", func() {
			It("should contain the routes", func() {
				sandboxNS, err := sandboxRepo.Get(sandboxName)
				Expect(err).NotTo(HaveOccurred())
				defer sandboxNS.Destroy()

				err = containerNamespace.Execute(func(_ *os.File) error {
					l, err := netlink.LinkByName("vx-eth0")
					Expect(err).NotTo(HaveOccurred())

					routes, err := netlink.RouteList(l, netlink.FAMILY_V4)
					Expect(err).NotTo(HaveOccurred())
					Expect(routes).To(HaveLen(3))

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
						Dst: vxlanNet,
						Gw:  ipamResult.IP4.Gateway.To4(),
					}))

					_, linkLocal, err := net.ParseCIDR("192.168.1.0/24")
					Expect(err).NotTo(HaveOccurred())
					Expect(sanitizedRoutes).To(ContainElement(netlink.Route{
						Dst: linkLocal,
						Src: ipamResult.IP4.IP.IP.To4(),
					}))

					_, dest, err := net.ParseCIDR("10.10.10.0/24")
					Expect(sanitizedRoutes).To(ContainElement(netlink.Route{
						Dst: dest,
						Gw:  ipamResult.IP4.Gateway.To4(),
					}))

					return nil
				})

				By("deleting")
				req, err := http.NewRequest("DELETE", deleteURL, bytes.NewReader(deletePayload))
				Expect(err).NotTo(HaveOccurred())

				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
			})
		})
	})
})
