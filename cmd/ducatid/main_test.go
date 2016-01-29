package main_test

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

var _ = Describe("Main", func() {
	var session *gexec.Session
	var address string

	BeforeEach(func() {
		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())
		ducatiCmd := exec.Command(ducatidPath, "-listenAddr", address)
		var err error
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Kill()
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

	It("should respond to GET /containers", func() {
		url := fmt.Sprintf("http://%s/containers", address)

		Eventually(serverIsAvailable).Should(Succeed())

		resp, err := http.Get(url)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		jsonBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(jsonBytes).To(MatchJSON("[]"))
	})

	Context("when there are containers", func() {
		var url string
		var addedContainers []models.Container

		BeforeEach(func() {
			url = fmt.Sprintf("http://%s/containers", address)

			Eventually(serverIsAvailable).Should(Succeed())

			addedContainers = []models.Container{
				{ID: "container-0-id"},
				{ID: "container-1-id"},
			}

			for _, container := range addedContainers {
				containerJSON, err := json.Marshal(container)
				Expect(err).NotTo(HaveOccurred())

				resp, err := http.Post(url, "application/json", bytes.NewReader(containerJSON))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
			}
		})

		It("can get a list of containers", func() {
			resp, err := http.Get(url)
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

		It("can retrieve a particular container", func() {
			url = fmt.Sprintf("http://%s/containers/%s", address, "container-0-id")
			resp, err := http.Get(url)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			jsonBytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var foundContainer models.Container
			err = json.Unmarshal(jsonBytes, &foundContainer)
			Expect(err).NotTo(HaveOccurred())
			Expect(foundContainer).To(Equal(addedContainers[0]))
		})

		It("can delete a container", func() {
			url = fmt.Sprintf("http://%s/containers/%s", address, "container-0-id")

			By("issuing a DELETE request")
			req, err := http.NewRequest("DELETE", url, nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			By("checking that the GET now returns a 404")
			resp, err = http.Get(url)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
