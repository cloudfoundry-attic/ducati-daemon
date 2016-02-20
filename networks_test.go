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

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = XDescribe("Networks", func() {
	var (
		session     *gexec.Session
		address     string
		networkID   string
		containerID string
	)

	BeforeEach(func() {
		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())
		ducatiCmd := exec.Command(ducatidPath, "-listenAddr", address, "-overlayNetwork", "192.168.0.0/16", "-localSubnet", "192.168.99.0/24")
		var err error
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		networkID = "some-network-id"
		containerID = "some-container-id"
	})

	AfterEach(func() {
		session.Kill()
		Eventually(session).Should(gexec.Exit())
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

	It("should respond to GET /networks/:network_id/containers with a list of container ids", func() {
		listURL := fmt.Sprintf("http://%s/%s", address, networkID)

		Eventually(serverIsAvailable).Should(Succeed())

		resp, err := http.Get(listURL)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		jsonBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(jsonBytes).To(MatchJSON("[]"))
	})

	Context("when there are containers in a network", func() {
		var listURL string
		var addedContainers []models.Container

		BeforeEach(func() {
			listURL = fmt.Sprintf("http://%s/%s", address, networkID)

			Eventually(serverIsAvailable).Should(Succeed())

			addedContainers = []models.Container{
				{ID: "container-0-id"},
				{ID: "container-1-id"},
			}

			for _, container := range addedContainers {
				containerJSON, err := json.Marshal(container)
				Expect(err).NotTo(HaveOccurred())

				createURL := fmt.Sprintf("http://%s/%s/%s", address, networkID, containerID)

				req, err := http.NewRequest("POST", createURL, bytes.NewReader(containerJSON))
				Expect(err).NotTo(HaveOccurred())

				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			}
		})

		It("can get a list of containers", func() {
			resp, err := http.Get(listURL)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			jsonBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var containers []models.Container
			err = json.Unmarshal(jsonBytes, &containers)
			Expect(err).NotTo(HaveOccurred())
			Expect(containers).To(ConsistOf(addedContainers))
		})
	})
})
