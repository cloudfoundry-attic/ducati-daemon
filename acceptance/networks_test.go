package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

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

		networkID = "some-network-id"
		containerID = "some-container-id"
	})

	AfterEach(func() {
		session.Kill()
		Eventually(session).Should(gexec.Exit())
		containerNamespace.Destroy()
	})

	It("should boot and gracefully terminate", func() {
		Consistently(session).ShouldNot(gexec.Exit())

		session.Interrupt()
		Eventually(session, 3*time.Second).Should(gexec.Exit(0))
	})

	var serverIsAvailable = func() error {
		_, err := net.Dial("tcp", address)
		return err
	}

	Describe("POST /networks/:network_id/:container_id", func() {
		var (
			createURL  string
			payload    []byte
			ipamResult types.Result
		)

		BeforeEach(func() {
			Eventually(serverIsAvailable).Should(Succeed())

			By("generating config and creating the request")
			ipamResult = types.Result{
				IP4: &types.IPConfig{
					IP: net.IPNet{
						IP:   net.ParseIP("192.168.100.2"),
						Mask: net.CIDRMask(24, 32),
					},
					Gateway: net.ParseIP("192.168.100.1"),
				},
			}

			var err error
			payload, err = json.Marshal(models.NetworksSetupContainerPayload{
				Args:               "FOO=BAR;ABC=123",
				ContainerNamespace: containerNamespace.Path(),
				InterfaceName:      "vx-eth0",
				VNI:                99,
				IPAM:               ipamResult,
			})
			Expect(err).NotTo(HaveOccurred())

			createURL = fmt.Sprintf("http://%s/networks/%s/%s", address, networkID, containerID)

		})

		It("should respond to POST /networks/:network_id/:container_id", func() {
			req, err := http.NewRequest("POST", createURL, bytes.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())

			By("creating the container")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			sandboxNS, err := sandboxRepo.Get("vni-99")
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

			By("getting the newly created container")
			listURL := fmt.Sprintf("http://%s/networks/%s", address, networkID)
			resp, err = http.Get(listURL)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			jsonBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var containers []models.Container
			err = json.Unmarshal(jsonBytes, &containers)
			Expect(err).NotTo(HaveOccurred())
			Expect(containers).To(HaveLen(1))
		})

		It("moves a vxlan adapter into the sandbox", func() {
			req, err := http.NewRequest("POST", createURL, bytes.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			sandboxNS, err := sandboxRepo.Get("vni-99")
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

			sandboxNS.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName("vxlan99")
				Expect(err).NotTo(HaveOccurred())
				vxlan, ok := link.(*netlink.Vxlan)
				Expect(ok).To(BeTrue())

				Expect(vxlan.VxlanId).To(Equal(99))
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

			req, err := http.NewRequest("POST", createURL, bytes.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			sandboxNS, err := sandboxRepo.Get("vni-99")
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

			err = sandboxNS.Execute(func(_ *os.File) error {
				link, err := netlink.LinkByName("vxlanbr99")
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

		It("creates a veth pair in the container and sandbox namespaces", func() {
			req, err := http.NewRequest("POST", createURL, bytes.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			sandboxNS, err := sandboxRepo.Get("vni-99")
			Expect(err).NotTo(HaveOccurred())
			defer sandboxNS.Destroy()

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
		})
	})
})
