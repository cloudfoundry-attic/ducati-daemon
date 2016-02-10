package acceptance_test

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const networkName = "some-network-name"

var _ = Describe("IP Address Management", func() {
	var session *gexec.Session
	var address string
	var daemonClient *client.DaemonClient

	BeforeEach(func() {
		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())
		ducatiCmd := exec.Command(ducatidPath, "-listenAddr", address, "-localSubnet", "192.168.3.0/30")
		var err error
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		daemonClient = client.New(fmt.Sprintf("http://%s", address), http.DefaultClient)
	})

	AfterEach(func() {
		session.Kill()
		Eventually(session).Should(gexec.Exit())
	})

	var serverIsAvailable = func() error {
		_, err := net.Dial("tcp", address)
		return err
	}

	It("should boot and gracefully terminate", func() {
		Eventually(serverIsAvailable).Should(Succeed())
		Consistently(session).ShouldNot(gexec.Exit())

		session.Interrupt()
		Eventually(session, 3*time.Second).Should(gexec.Exit(0))
	})

	Describe("POST", func() {
		It("should respond to /ipam/:network_id/:container_id", func() {
			Eventually(serverIsAvailable).Should(Succeed())

			ipamResult, err := daemonClient.AllocateIP("some-network", "some-container")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipamResult.IP4.IP.String()).To(Equal("192.168.3.2/30"))
			Expect(ipamResult.IP4.Gateway.String()).To(Equal("192.168.3.1"))
		})
	})
})
