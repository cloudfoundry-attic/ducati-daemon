package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Networks", func() {
	var (
		session     *gexec.Session
		address     string
		networkID   string
		containerID string

		containerNamespace namespace.Namespace
	)

	BeforeEach(func() {
		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())
		sandboxRepoDir, err := ioutil.TempDir("", "sandbox")
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

	It("should respond to POST /networks/:network_id/:container_id", func() {
		Eventually(serverIsAvailable).Should(Succeed())

		By("generating config and creating the request")
		ipamResult := types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("192.168.100.2"),
					Mask: net.CIDRMask(24, 32),
				},
				Gateway: net.ParseIP("192.168.100.1"),
			},
		}

		payload, err := json.Marshal(models.NetworksSetupContainerPayload{
			Args:               "FOO=BAR;ABC=123",
			ContainerNamespace: containerNamespace.Path(),
			InterfaceName:      "interface-name",
			VNI:                99,
			IPAM:               ipamResult,
		})

		createURL := fmt.Sprintf("http://%s/networks/%s/%s", address, networkID, containerID)

		req, err := http.NewRequest("POST", createURL, bytes.NewReader(payload))
		Expect(err).NotTo(HaveOccurred())

		By("creating the container")
		resp, err := http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusCreated))

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
})
