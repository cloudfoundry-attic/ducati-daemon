package acceptance_test

import (
	"os/exec"
	"strings"

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

func replaceFlag(src []string, key string, newValue string) []string {
	replaced := []string{}
	didReplace := false
	for _, element := range src {
		if strings.HasPrefix(element, "-"+key+"=") {
			replaced = append(replaced, "-"+key+"="+newValue)
			didReplace = true
		} else {
			replaced = append(replaced, element)
		}
	}

	if !didReplace {
		Fail("test setup error: didn't find expected flag")
	}

	return replaced
}

var _ = Describe("Ducati Daemon Flag Validation", func() {
	var (
		session *gexec.Session
		err     error
		flags   []string
	)

	BeforeEach(func() {
		flags = []string{
			"-listenAddr=some-listen-address",
			"-overlayNetwork=192.168.0.0/16",
			"-localSubnet=192.168.0.1/24",
			"-databaseURL=some-database-url",
			"-sandboxRepoDir=/some/path",
		}
	})

	AfterEach(func() {
		if session != nil {
			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		}
	})

	DescribeTable("flag errors",
		func(expectedError, flagKey, flagValue string) {
			brokenFlags := replaceFlag(flags, flagKey, flagValue)
			session, err = startDaemon(brokenFlags...)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say(expectedError))
		},

		Entry("missing listenAddr",
			`missing required flag "listenAddr"`, "listenAddr", ""),

		Entry("missing overlayNetwork flag",
			`missing required flag "overlayNetwork"`, "overlayNetwork", ""),

		Entry("missing localSubnet flag",
			`missing required flag "localSubnet"`, "localSubnet", ""),

		Entry("missing databaseURL",
			`missing required flag "databaseURL"`, "databaseURL", ""),

		Entry("missing sandboxRepoDir",
			`missing required flag "sandboxRepoDir"`, "sandboxRepoDir", ""),

		Entry("overlayNetwork does not contain localSubnet",
			`overlay network does not contain local subnet`, "overlayNetwork", "192.168.3.0/28"),

		Entry("localSubnet is not a valid CIDR",
			`invalid CIDR provided for "localSubnet": gobbledygook`, "localSubnet", "gobbledygook"),

		Entry("overlayNetwork is not a valid CIDR",
			`invalid CIDR provided for "overlayNetwork": gobbledygook`, "overlayNetwork", "gobbledygook"),

		Entry("overlayNetwork does not contain localSubnet",
			`overlay network does not contain local subnet`, "overlayNetwork", "192.168.3.0/28"),

		Entry("invalid database url",
			`missing required flag "databaseURL"`, "databaseURL", ""),
	)
})
