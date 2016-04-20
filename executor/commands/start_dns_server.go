package commands

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type StartDNSServer struct {
	Namespace     namespace.Namespace
	ListenAddress string
	SandboxName   string
}

func (sd StartDNSServer) Execute(context executor.Context) error {
	listenerFactory := context.ListenerFactory()

	var conn *net.UDPConn
	err := sd.Namespace.Execute(func(*os.File) error {
		var err error
		conn, err = listenerFactory.ListenUDP("udp", sd.ListenAddress)
		return err
	})
	if err != nil {
		return fmt.Errorf("listen udp: %s", err)
	}

	dnsServerRunner, err := context.DNSServerFactory().New(conn)
	if err != nil {
		return fmt.Errorf("new dns server: %s", err)
	}

	sbox, err := context.SandboxRepository().Get(sd.SandboxName)
	if err != nil {
		return fmt.Errorf("get sandbox: %s", err)
	}

	err = sbox.LaunchDNS(dnsServerRunner)
	if err != nil {
		return fmt.Errorf("sandbox launch dns: %s", err)
	}

	return nil
}

func (sd StartDNSServer) String() string {
	return ""
}
