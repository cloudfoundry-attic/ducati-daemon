package acceptance_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os/exec"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("DB Connection Retry", func() {
	var (
		address string
		session *gexec.Session

		proxy        *Proxy
		databaseHost string
	)

	BeforeEach(func() {
		databaseURL, err := url.Parse(testDatabase.URL())
		Expect(err).NotTo(HaveOccurred())

		databaseHost = databaseURL.Host

		proxyURL := databaseURL
		proxyURL.Host = fmt.Sprintf("127.0.0.1:%d", 14001+GinkgoParallelNode())

		proxy = &Proxy{
			Host: proxyURL.Host,
		}

		address = fmt.Sprintf("127.0.0.1:%d", 4001+GinkgoParallelNode())

		sandboxRepoDir, err := ioutil.TempDir("", "sandbox")
		Expect(err).NotTo(HaveOccurred())

		ducatiCmd := exec.Command(
			ducatidPath,
			"-listenAddr", address,
			"-overlayNetwork", "192.168.0.0/16",
			"-localSubnet", "192.168.99.0/24",
			"-databaseURL", proxyURL.String(),
			"-sandboxRepoDir", sandboxRepoDir,
		)
		session, err = gexec.Start(ducatiCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		proxy.Close()

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
	})

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	Context("when the daemon starts before the database is up", func() {
		It("waits for the database to start before accepting connections", func() {
			Consistently(serverIsAvailable).ShouldNot(Succeed())
			Consistently(session).ShouldNot(gexec.Exit())

			go proxy.Proxy(databaseHost)

			Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
		})
	})
})

type Proxy struct {
	sync.Mutex
	Host     string
	Listener net.Listener
}

func (p *Proxy) Proxy(targetHost string) {
	l, err := net.Listen("tcp", p.Host)
	if err != nil {
		fmt.Printf("listen failed: %s", err)
		return
	}

	p.Lock()
	p.Listener = l
	p.Unlock()

	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}

		go p.handleConnection(targetHost, conn)
	}
}

func (p *Proxy) Close() error {
	p.Lock()
	defer p.Unlock()
	return p.Listener.Close()
}

func (p *Proxy) handleConnection(targetHost string, conn net.Conn) {
	dbConn, err := net.Dial("tcp", targetHost)
	if err != nil {
		fmt.Printf("dial: %s", err)
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() { io.Copy(dbConn, conn); dbConn.Close(); wg.Done() }()
	go func() { io.Copy(conn, dbConn); conn.Close(); wg.Done() }()

	wg.Wait()
}
