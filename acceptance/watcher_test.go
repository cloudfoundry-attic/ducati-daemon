package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
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

		upSpec       models.NetworksSetupContainerPayload
		downSpec     models.NetworksDeleteContainerPayload
		daemonClient *client.DaemonClient
		ipamResult   types.Result
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
			"-localSubnet", "192.168.1.0/24",
			"-databaseURL", testDatabase.URL(),
			"-sandboxRepoDir", sandboxRepoDir,
		)
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		networkID = fmt.Sprintf("some-network-id-%x", rand.Int())
		containerID = fmt.Sprintf("some-container-id-%x", rand.Int())
		vni = GinkgoParallelNode() // necessary to avoid test pollution in parallel
		sandboxName = fmt.Sprintf("vni-%d", vni)

		var serverIsAvailable = func() error {
			return VerifyTCPConnection(address)
		}
		Eventually(serverIsAvailable).Should(Succeed())

		daemonClient = client.New("http://"+address, http.DefaultClient)

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

		upSpec = models.NetworksSetupContainerPayload{
			Args:               "FOO=BAR;ABC=123",
			ContainerNamespace: containerNamespace.Path(),
			InterfaceName:      "vx-eth0",
			VNI:                vni,
			IPAM:               ipamResult,
		}

		downSpec = models.NetworksDeleteContainerPayload{
			InterfaceName:      "vx-eth0",
			ContainerNamespace: containerNamespace.Path(),
			VNI:                vni,
		}

		By("adding the container to a network")
		Expect(daemonClient.ContainerUp(networkID, containerID, upSpec)).To(Succeed())
	})

	AfterEach(func() {
		By("removing the container from the network")
		Expect(daemonClient.ContainerDown(networkID, containerID, downSpec)).To(Succeed())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		containerNamespace.Destroy()
	})

	It("catches L3 misses", func() {
		time.Sleep(2 * time.Second)
		err := containerNamespace.Execute(func(_ *os.File) error {
			_, err := net.DialTimeout("tcp", "192.168.1.100:1234", 1*time.Second)
			Expect(err).To(HaveOccurred())
			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		Eventually(session.Out, "5s").Should(gbytes.Say("STALE.*IP: 192.168.1.100"))
	})
})
