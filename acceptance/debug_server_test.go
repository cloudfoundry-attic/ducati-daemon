package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	"github.com/cloudfoundry-incubator/ducati-daemon/config"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Debug Server Test", func() {
	var (
		address, debugAddress string
		hostAddress           string
		session               *gexec.Session

		logger             *lagertest.TestLogger
		sandboxRepo        namespace.Repository
		containerRepo      namespace.Repository
		containerNamespace namespace.Namespace
	)

	BeforeEach(func() {
		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())
		debugAddress = fmt.Sprintf("127.0.0.1:%d", 19000+GinkgoParallelNode())

		sandboxRepoDir, err := ioutil.TempDir("", "sandbox")
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")

		sandboxRepo, err = namespace.NewRepository(logger, sandboxRepoDir)
		Expect(err).NotTo(HaveOccurred())

		containerRepoDir, err := ioutil.TempDir("", "containers")
		Expect(err).NotTo(HaveOccurred())

		containerRepo, err = namespace.NewRepository(logger, containerRepoDir)
		Expect(err).NotTo(HaveOccurred())

		guid, err := uuid.NewV4()
		Expect(err).NotTo(HaveOccurred())

		containerNamespace, err = containerRepo.Create(guid.String())
		Expect(err).NotTo(HaveOccurred())

		hostAddress = "10.11.12.13"

		configFilePath := writeConfigFile(config.Daemon{
			ListenHost:        "127.0.0.1",
			ListenPort:        4001 + GinkgoParallelNode(),
			LocalSubnet:       "192.168.1.0/16",
			OverlayNetwork:    "192.168.0.0/16",
			SandboxDir:        sandboxRepoDir,
			Database:          testDatabase.DBConfig(),
			HostAddress:       hostAddress,
			OverlayDNSAddress: "192.168.255.254",
			ExternalDNSServer: "8.8.8.8",
			Suffix:            "potato",
			DebugAddress:      debugAddress,
		})

		ducatiCmd := exec.Command(ducatidPath, "-configFile", configFilePath)
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(containerRepo.Destroy(containerNamespace)).To(Succeed())
	})

	var debugServerIsAvailable = func() error {
		return VerifyTCPConnection(debugAddress)
	}

	It("starts the debug server on the specified address", func() {
		Eventually(debugServerIsAvailable).Should(Succeed())
		Consistently(session).ShouldNot(gexec.Exit())

		resp, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/cmdline", debugAddress))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		bytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(bytes).To(ContainSubstring("configFile"))
	})
})
