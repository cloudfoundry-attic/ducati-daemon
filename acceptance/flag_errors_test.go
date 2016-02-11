package acceptance_test

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func startDaemon(args ...string) (*gexec.Session, error) {
	ducatiCmd := exec.Command(ducatidPath, args...)
	return gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
}

var _ = Describe("Ducati Daemon Flag Validation", func() {
	var (
		address string
		session *gexec.Session
		err     error
	)

	BeforeEach(func() {
		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())
	})

	AfterEach(func() {
		if session != nil {
			session.Kill()
			Eventually(session).Should(gexec.Exit())
		}
	})

	Context("when the listenAddr flag is missing", func() {
		It("fails to start and provides a meaninful error", func() {
			session, err = startDaemon("-localSubnet=192.168.3.0/16", "-overlayNetwork=192.168.0.0/16")
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(`missing required flag "listenAddr"`))
		})
	})

	Context("when the overlayNetwork flag is missing", func() {
		It("fails to start and provides a meaninful error", func() {
			session, err = startDaemon("-listenAddr", address, "-localSubnet=192.168.3.0/16")
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(`missing required flag "overlayNetwork"`))
		})
	})

	Context("when the localSubnet flag is missing", func() {
		It("fails to start and provides a meaninful error", func() {
			session, err = startDaemon("-listenAddr", address, "-overlayNetwork=192.168.3.0/16")
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(`missing required flag "localSubnet"`))
		})
	})

	Context("when the overlayNetwork does not contain localSubnet", func() {
		It("fails to start and provides a meaninful error", func() {
			session, err = startDaemon("-listenAddr", address, "-overlayNetwork=192.168.3.0/24", "-localSubnet=192.168.4.0/24")
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(`overlay network does not contain local subnet`))
		})
	})

	Context("when localSubnet is not a valid CIDR", func() {
		It("fails to start and provides a meaninful error", func() {
			session, err = startDaemon("-listenAddr", address, "-overlayNetwork=192.168.3.0/24", "-localSubnet=gobbledygook")
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(`invalid CIDR provided for "localSubnet": gobbledygook`))
		})
	})

	Context("when overlayNetwork is not a valid CIDR", func() {
		It("fails to start and provides a meaninful error", func() {
			session, err = startDaemon("-listenAddr", address, "-overlayNetwork=gobbledygook", "-localSubnet=192.168.1.0/24")
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(`invalid CIDR provided for "overlayNetwork": gobbledygook`))
		})
	})
})
