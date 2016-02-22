package acceptance_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
		session *gexec.Session
		err     error
	)

	AfterEach(func() {
		if session != nil {
			session.Kill()
			Eventually(session).Should(gexec.Exit())
		}
	})

	DescribeTable("flag errors",
		func(expectedError string, flags ...string) {
			session, err = startDaemon(flags...)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(expectedError))
		},
		Entry("missing listenAddr", `missing required flag "listenAddr"`,
			"-localSubnet=192.168.3.0/16", "-overlayNetwork=192.168.0.0/16"),

		Entry("missing overlayNetwork flag", `missing required flag "listenAddr"`,
			"-localSubnet=192.168.3.0/16", "-overlayNetwork=192.168.0.0/16"),

		Entry("missing localSubnet flag", `missing required flag "localSubnet"`,
			"-listenAddr=some-listen-address", "-overlayNetwork=192.168.3.0/16"),

		Entry("overlayNetwork does not container localSubnet", `overlay network does not contain local subnet`,
			"-listenAddr=some-listen-address", "-overlayNetwork=192.168.3.0/24", "-localSubnet=192.168.4.0/24"),

		Entry("localSubnet is not a valid CIDR", `invalid CIDR provided for "localSubnet": gobbledygook`,
			"-listenAddr=some-listen-address", "-overlayNetwork=192.168.3.0/24", "-localSubnet=gobbledygook"),

		Entry("overlayNetwork is not a valid CIDR", `invalid CIDR provided for "overlayNetwork": gobbledygook`,
			"-listenAddr=some-listen-address", "-overlayNetwork=gobbledygook", "-localSubnet=192.168.1.0/24"),
	)
})
