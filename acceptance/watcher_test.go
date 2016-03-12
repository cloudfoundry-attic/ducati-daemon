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

	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/config"
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

		configFilePath := writeConfigFile(config.Daemon{
			ListenHost:     "127.0.0.1",
			ListenPort:     4001 + GinkgoParallelNode(),
			LocalSubnet:    "192.168.1.0/24",
			OverlayNetwork: "192.168.0.0/16",
			HostAddress:    "10.11.12.13",
			SandboxDir:     sandboxRepoDir,
			Database:       testDatabase.AsDaemonConfig(),
		})

		ducatiCmd := exec.Command(ducatidPath, "-configFile", configFilePath)
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
		upSpec = models.NetworksSetupContainerPayload{
			Args:               "FOO=BAR;ABC=123",
			ContainerNamespace: containerNamespace.Path(),
			InterfaceName:      "vx-eth0",
			VNI:                vni,
		}

		downSpec = models.NetworksDeleteContainerPayload{
			InterfaceName:      "vx-eth0",
			ContainerNamespace: containerNamespace.Path(),
			VNI:                vni,
		}

		By("adding the container to a network")
		_, err = daemonClient.ContainerUp(networkID, containerID, upSpec)
		Expect(err).NotTo(HaveOccurred())
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

		Eventually(session.Out, "5s").Should(gbytes.Say("ducati-d.sandbox-miss.*dest_ip.*192.168.1.100.*sandbox.*vni-%d", vni))
	})
})
