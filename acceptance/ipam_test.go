package acceptance_test

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
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
		ducatiCmd := exec.Command(
			ducatidPath,
			"-listenAddr", address,
			"-overlayNetwork", "192.168.0.0/16",
			"-localSubnet", "192.168.3.0/30",
			"-databaseURL", testDatabase.URL(),
		)
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
		return VerifyTCPConnection(address)
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

	Describe("DELETE", func() {
		It("should respond to /ipam/:network_id/:container_id", func() {
			Eventually(serverIsAvailable).Should(Succeed())

			err := daemonClient.ReleaseIP("some-network", "some-container")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("address exhaustion", func() {
		It("should run out of addresses", func() {
			Eventually(serverIsAvailable).Should(Succeed())

			firstResult, err := daemonClient.AllocateIP("some-network", "some-container-1")
			Expect(err).NotTo(HaveOccurred())

			Expect(firstResult.IP4.IP.String()).To(Equal("192.168.3.2/30"))
			Expect(firstResult.IP4.Gateway.String()).To(Equal("192.168.3.1"))

			secondResult, err := daemonClient.AllocateIP("some-network", "some-container-2")
			Expect(err).NotTo(HaveOccurred())

			Expect(secondResult.IP4.IP.String()).To(Equal("192.168.3.3/30"))
			Expect(secondResult.IP4.Gateway.String()).To(Equal("192.168.3.1"))

			_, err = daemonClient.AllocateIP("some-network", "some-container-2")
			Expect(err).To(Equal(ipam.NoMoreAddressesError))

			err = daemonClient.ReleaseIP("some-network", "some-container-2")
			Expect(err).NotTo(HaveOccurred())

			tryAgainResult, err := daemonClient.AllocateIP("some-network", "some-container-2")
			Expect(err).NotTo(HaveOccurred())

			Expect(tryAgainResult.IP4.IP.String()).To(Equal("192.168.3.3/30"))
			Expect(tryAgainResult.IP4.Gateway.String()).To(Equal("192.168.3.1"))
		})
	})
})
