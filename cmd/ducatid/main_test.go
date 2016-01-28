package main_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"time"

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

		jsonBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(jsonBytes).To(MatchJSON("[]"))
	})

})
